package main

import (
    "bufio"
    "os/exec"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
)


type cGroupPath struct {
    ctype   string
    Cpath   string
}


func getCgroups(pid string) ([]cGroupPath, error) {
    fmt.Println("Reading the cgroups at:", fmt.Sprintf("/proc/%s/cgroup", pid))
    listOfCgroups := []cGroupPath{}
    cGroupPrefix := "/sys/fs/cgroup/"
    fd, err := os.Open(filepath.Join("/proc", pid, "cgroup"))
    defer fd.Close()

    if err != nil {
        fmt.Println("Could not read the cgroup file: ", fmt.Sprintf("/proc/%s/cgroup", pid), " because of ", err)
        return listOfCgroups, err
    }

    // Read the file line by line
    scanner := bufio.NewScanner(fd)
    scanner.Split(bufio.ScanLines)

    for scanner.Scan() {
        // Split the line by ":"
        s := strings.Split(scanner.Text(), ":")

        listOfCgroups = append(listOfCgroups, cGroupPath{ctype: s[1], Cpath: cGroupPrefix+s[1]+s[2]+"/tasks"})
    }

    return listOfCgroups, nil
}


func appendToFile(data string, path string) error {
    fmt.Println("Writing the pid to this file", path)
    // Open the file for writing
    f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }

    _, err1 := f.WriteString(data)
    if err1 != nil {
        return err
    }

    // The write was successful
    return nil
}


func main() {
    if len(os.Args) != 2 {
        fmt.Println("abort: you must provide a PID as the sole argument")
        os.Exit(1)
    }

    targetPid := os.Args[1]
    selfPid := os.Getpid()

    listOfCgroups, _ := getCgroups(targetPid)

    // Add our pid to all the target cgroups
    for _, c := range listOfCgroups {
        appendToFile(strconv.Itoa(selfPid), c.Cpath)
    }

    // Now, enter into the target namespaces
    targetNamespaces := []string{"ipc", "uts", "net", "pid", "mnt"}

    for i := range targetNamespaces {
        fd, _ := syscall.Open(filepath.Join("/proc", targetPid, "ns", targetNamespaces[i]), syscall.O_RDONLY, 0644)
        err, _, msg := syscall.RawSyscall(308, uintptr(fd), 0, 0) // 308 == setns

        if err != 0 {
            fmt.Println("setns on", targetNamespaces[i], "namespace failed:", msg)
        } else {
            fmt.Println("setns on", targetNamespaces[i], "namespace succeeded")
        }

    }

    fmt.Println("String a new process.......")

    // After setting cgroups and namespaces, fork-exec the binary provided by the user.
    cmd := exec.Command("/bin/sleep", "120")
    cmd.Run()
}
