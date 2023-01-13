package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	mapset "github.com/deckarep/golang-set/v2"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table         table.Model
	outdated      []outdatedDep
	toUpdate      mapset.Set[string]
	shouldInstall bool
	done          chan bool
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case " ":
			thisDep := m.table.SelectedRow()[1]
			if m.toUpdate.Contains(thisDep) {
				m.toUpdate.Remove(thisDep)
			} else {
				m.toUpdate.Add(thisDep)
			}

			m.table.SetRows(renderRows(m.outdated, m.toUpdate))
			return m, nil
		case "enter":
			m.shouldInstall = true
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) Install() {
	depsToArgs := func(deps []string) []string {
		args := make([]string, 0)
		for _, depName := range deps {
			args = append(args, depName+"@latest")
		}
		return args
	}

	isDevDep := func(dep string) bool {
		for _, outdated := range m.outdated {
			if outdated.Name == dep {
				return outdated.IsDev
			}
		}
		return false
	}

	runInstall := func(deps []string, isDev bool) {
		fmt.Print("\n\n> npm ")
		args := []string{"i"}
		if isDev {
			args = append(args, "-D")
		}
		args = append(args, depsToArgs(deps)...)
		fmt.Println(strings.Join(args, " "))

		cmd := exec.Command("npm", args...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Run()

		if err != nil {
			fmt.Println("Err: Error occurred updating packages", err)
			os.Exit(1)
		}
	}

	deps := make([]string, 0)
	devDeps := make([]string, 0)

	for dep := range m.toUpdate.Iterator().C {
		if isDevDep(dep) {
			devDeps = append(devDeps, dep)
		} else {
			deps = append(deps, dep)
		}
	}

	if len(deps) > 0 {
		runInstall(deps, false)
	}
	if len(devDeps) > 0 {
		runInstall(devDeps, true)
	}
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

type depVersions struct {
	Current string `json:"current"`
	Wanted  string `json:"wanted"`
	Latest  string `json:"latest"`
}

type npmOutdated map[string]depVersions

type outdatedDep struct {
	Name, Current, Latest string
	IsDev                 bool
}

type packageJson struct {
	Dependencies map[string]string `json:"dependencies"`
}

func getPackageJsonDeps() mapset.Set[string] {
	deps := mapset.NewSet[string]()

	content, err := ioutil.ReadFile("./package.json")
	if err != nil {
		fmt.Println("Err: Couldn't read package.json", err)
		os.Exit(1)
	}

	pj := packageJson{map[string]string{}}

	err = json.Unmarshal(content, &pj)
	if err != nil {
		fmt.Println("Err: Couldn't parse package.json", err)
		os.Exit(1)
	}

	for name := range pj.Dependencies {
		deps.Add(name)
	}

	return deps
}

func getOutdated() []outdatedDep {
	// ignore errors, if deps are outdated the exit code is 1, but that is fine for us
	out, _ := exec.Command("npm", "outdated", "--json").Output()

	versions := make(npmOutdated)
	err := json.Unmarshal(out, &versions)

	if err != nil {
		log.Fatal(string(out))
	}

	depsSet := getPackageJsonDeps()

	outdated := make([]outdatedDep, 0)
	for name, dep := range versions {
		outdated = append(outdated, outdatedDep{
			Name:    name,
			Current: dep.Current,
			Latest:  dep.Latest,
			IsDev:   !depsSet.Contains(name),
		})
	}

	sort.Slice(outdated, func(i, j int) bool {
		return outdated[i].Name < outdated[j].Name
	})

	return outdated
}

func renderRows(outdated []outdatedDep, toUpdate mapset.Set[string]) []table.Row {
	rows := make([]table.Row, 0)
	for _, dep := range outdated {
		selected := ""
		if toUpdate != nil && toUpdate.Contains(dep.Name) {
			selected = "x"
		}

		isDev := ""
		if dep.IsDev {
			isDev = "x"
		}
		rows = append(rows, table.Row{selected, dep.Name, dep.Current, dep.Latest, isDev})
	}

	return rows
}

func main() {
	columns := []table.Column{
		{Title: "?", Width: 4},
		{Title: "Name", Width: 30},
		{Title: "Current", Width: 10},
		{Title: "Latest", Width: 10},
		{Title: "Dev Dep", Width: 7},
	}

	outdated := getOutdated()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(renderRows(outdated, nil)),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	t.SetHeight(30)

	m := model{t, outdated, mapset.NewSet[string](), false, make(chan bool)}

	if _, err := tea.NewProgram(&m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m.shouldInstall {
		m.Install()
	}
}
