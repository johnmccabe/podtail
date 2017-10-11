package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var since string
var tail string
var regexType string
var namespace string
var context string
var selector string
var versionFlag bool
var kubeconfig string
var kubectl string

// Version of the podtail binary
var Version string

// Execute initialises Cobra
func Execute() {
	rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&since, "since", "s", "10s", "Only return logs newer than a relative duration like 5s, 2m, or 3h.")
	rootCmd.PersistentFlags().StringVar(&tail, "tail", "-1", "Lines of recent log file to display. -1 shows all lines.")
	rootCmd.PersistentFlags().StringVarP(&regexType, "regex", "e", "substring", "The type of name matching to use (regex|substring).")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "The Kubernetes namespace where the pods are located.")
	rootCmd.PersistentFlags().StringVarP(&context, "context", "t", "", "The k8s context. ex. int1-context. Relies on ~/.kube/config for the contexts.")
	rootCmd.PersistentFlags().StringVarP(&selector, "selector", "l", "", "Label selector. If used the pod name is ignored.")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Prints the kubetail version.")

	// Flags to enable running with kubectl config and binaries in non-standard locations,
	// these flags do not have equivalents in kubetail.
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file.")
	defaultKubectl := "kubectl"
	if runtime.GOOS == "windows" {
		defaultKubectl = "kubectl.exe"
	}
	rootCmd.PersistentFlags().StringVar(&kubectl, "kubectl", defaultKubectl, "Path to the kubectl executable. Override with an explicit path if necessary.")
}

var rootCmd = &cobra.Command{
	Use:   "podtail SEARCH_TERM",
	Short: "Tail Kubernetes logs from multiple pods at the same time",
	Args:  cobra.MaximumNArgs(1),
	Run:   runPodtail,
}

type tailInfo struct {
	pod       string
	container string
	since     string
	tail      string
	context   string
	namespace string
	logColor  *color.Color
}

func runPodtail(cmd *cobra.Command, args []string) {

	if versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	var searchTerm string
	if len(args) > 0 {
		searchTerm = args[0]
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	c := logColor{}

	pods, err := getPods(searchTerm, context, namespace, selector, regexType)
	if err != nil {
		log.Fatal(err)
	}

	tails := []tailInfo{}

	for _, pod := range pods {
		logColor := color.New(c.next())
		logColor.Println(pod)
		containers, err := getContainers(pod, context, namespace)
		if err != nil {
			log.Fatal(err)
		}
		for _, container := range containers {
			t := tailInfo{
				pod:       pod,
				container: container,
				since:     since,
				tail:      tail,
				context:   context,
				namespace: namespace,
				logColor:  logColor,
			}
			tails = append(tails, t)
		}
	}

	if len(tails) == 0 {
		fmt.Println("No matching pods or containers detected. Exiting...")
		os.Exit(0)
	}

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	for _, t := range tails {
		go tailContainer(t.pod, t.container, t.since, t.tail, t.context, t.namespace, t.logColor)
	}

	<-done
	fmt.Println("exiting...")
}

func getPods(searchTerm, context, namespace, selector, regexType string) ([]string, error) {
	var args []string
	var pods []string
	var pattern string

	switch regexType {
	case "regex":
		pattern = searchTerm
	case "substring":
	default:
		fmt.Printf("Invalid regex type supplied: %s\n", regexType)
		os.Exit(1)
	}

	args = append(args, []string{"get", "pods"}...)
	args = append(args, fmt.Sprintf("--context=%s", context))
	args = append(args, fmt.Sprintf("--namespace=%s", namespace))
	args = append(args, "--output=jsonpath={.items[*].metadata.name}")

	if len(selector) > 0 {
		args = append(args, []string{"--selector", selector}...)
	}

	if len(kubeconfig) > 0 {
		args = append(args, fmt.Sprintf("--kubeconfig=%s", kubeconfig))
	}

	cmd := exec.Command(kubectl, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error running: %s\n", strings.Join(cmd.Args, " "))
		fmt.Println(stderr.String())
		return nil, err
	}

	pods = strings.Split(stdout.String(), " ")
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
			if strings.Contains(p, searchTerm) {
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
	args = append(args, fmt.Sprintf("--context=%s", context))
	args = append(args, fmt.Sprintf("--namespace=%s", namespace))
	args = append(args, "--output=jsonpath={.spec.containers[*].name}")

	if len(kubeconfig) > 0 {
		args = append(args, fmt.Sprintf("--kubeconfig=%s", kubeconfig))
	}

	cmd := exec.Command(kubectl, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error running: %s\n", strings.Join(cmd.Args, " "))
		fmt.Println(stderr.String())
		return nil, err
	}

	containers = strings.Split(stdout.String(), " ")

	return containers, nil
}

func tailContainer(pod, container, since, tail, context, namespace string, logColor *color.Color) error {
	var args []string

	args = append(args, fmt.Sprintf("--context=%s", context))
	args = append(args, "logs", pod, container, "-f")
	args = append(args, fmt.Sprintf("--since=%s", since))
	args = append(args, fmt.Sprintf("--tail=%s", tail))
	args = append(args, fmt.Sprintf("--namespace=%s", namespace))

	if len(kubeconfig) > 0 {
		args = append(args, fmt.Sprintf("--kubeconfig=%s", kubeconfig))
	}

	cmd := exec.Command(kubectl, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("error creating tail stdout pipe: %v\n", err)
		return err
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("error running: %s\n", strings.Join(cmd.Args, " "))
		return err
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
	color.FgCyan,
	color.FgHiRed,
	color.FgHiGreen,
	color.FgHiYellow,
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
