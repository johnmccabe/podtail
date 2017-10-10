package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

const kubectl = "kubectl"
const defaultSince = "10s"
const defaultTail = "-1"

func main() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	c := logColor{}

	// Match pod prefix
	pods, err := getPods("", "", "", "", "")
	// Match pod via regex
	// out, err := getPods("", "", "", ".*cont", "")
	if err != nil {
		log.Fatal(err)
	}

	tails := []func(){}

	for _, pod := range pods {
		logColor := color.New(c.next())
		logColor.Println(pod)
		containers, err := getContainers(pod, "", "")
		if err != nil {
			log.Fatal(err)
		}
		for _, container := range containers {
			tails = append(tails, func() { go tailContainer(pod, container, defaultSince, defaultTail, "", "", logColor) })
		}
	}

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	<-done
	fmt.Println("exiting...")
}

func getPods(pod, context, namespace, selector, pattern string) ([]string, error) {
	var args []string
	var pods []string

	args = append(args, []string{"get", "pods"}...)
	args = append(args, fmt.Sprintf("--context=%x", context))

	if len(selector) > 0 {
		args = append(args, []string{"--selector", selector}...)
	}

	args = append(args, fmt.Sprintf("--namespace=%s", namespace))
	args = append(args, "--output=jsonpath={.items[*].metadata.name}")

	cmd := exec.Command(kubectl, args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	pods = strings.Split(out.String(), " ")
	filtered := pods[:0]

	if len(pattern) > 0 {
		podMatch := regexp.MustCompilePOSIX(pattern)
		for _, p := range pods {
			if podMatch.MatchString(p) {
				filtered = append(filtered, p)
			}
		}
	} else {
		for _, p := range pods {
			if strings.Contains(p, pod) {
				filtered = append(filtered, p)
			}
		}
	}

	return filtered, nil
}

func getContainers(pod, context, namespace string) ([]string, error) {
	var args []string
	var containers []string

	args = append(args, []string{"get", "pod", pod}...)
	args = append(args, fmt.Sprintf("--context=%x", context))
	args = append(args, fmt.Sprintf("--namespace=%s", namespace))
	args = append(args, "--output=jsonpath={.spec.containers[*].name}")

	cmd := exec.Command(kubectl, args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	containers = strings.Split(out.String(), " ")

	return containers, nil
}

func tailContainer(pod, container, since, tail, context, namespace string, logColor *color.Color) error {
	var args []string

	args = append(args, fmt.Sprintf("--context=%x", context))
	args = append(args, "logs", pod, container, "-f")
	args = append(args, fmt.Sprintf("--since=%s", since))
	args = append(args, fmt.Sprintf("--tail=%s", tail))
	args = append(args, fmt.Sprintf("--namespace=%s", namespace))

	cmd := exec.Command(kubectl, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		logColor.Printf("[%s] %s\n", pod, scanner.Text())
	}

	return nil
}

var availableColours = [...]color.Attribute{
	color.FgRed,
	color.FgGreen,
	color.FgYellow,
	color.FgBlue,
	color.FgMagenta,
	color.FgCyan,
	color.FgHiRed,
	color.FgHiGreen,
	color.FgHiYellow,
	color.FgHiBlue,
	color.FgHiMagenta,
	color.FgHiCyan,
}

type logColor struct {
	index int
}

func (p *logColor) next() color.Attribute {
	c := availableColours[p.index]
	p.index = (p.index + 1) % len(availableColours)
	return c
}
