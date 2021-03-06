package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"syscall"

	"github.com/gdamore/tcell"
	"github.com/gofrs/uuid"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v2"
)

var (
	errEmptyYaml  = errors.New("empty yaml")
	colorDarkGrey = tcell.NewRGBColor(29, 32, 33)
)

type yamlType int

const (
	Root yamlType = iota + 1
	Object
	Array
	Key
	Value
)

var yamlTypeMap = map[yamlType]string{
	Object: "object",
	Array:  "array",
	Key:    "key",
	Value:  "value",
}

func (t yamlType) String() string {
	return yamlTypeMap[t]
}

type valueType int

const (
	Int valueType = iota + 1
	String
	Float
	Boolean
	Null
)

var valueTypeMap = map[valueType]string{
	Int:     "int",
	String:  "string",
	Float:   "float",
	Boolean: "boolean",
	Null:    "null",
}

func (v valueType) String() string {
	return valueTypeMap[v]
}

type Reference struct {
	ID        string
	YAMLType  yamlType
	ValueType valueType
}

func unmarshalYAML(in io.Reader) (interface{}, error) {
	b, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errEmptyYaml
	}

	var i interface{}
	if err := yaml.Unmarshal(b, &i); err != nil {
		return nil, err
	}

	return i, nil
}

type yq struct {
	tree  *tree
	app   *tview.Application
	pages *tview.Pages
}

func newYQ() *yq {
	tview.Styles.PrimitiveBackgroundColor = colorDarkGrey
	tview.Styles.PrimaryTextColor = tcell.NewRGBColor(235, 219, 178)
	g := &yq{
		tree:  newTree(),
		app:   tview.NewApplication(),
		pages: tview.NewPages(),
	}
	return g
}

func (g *yq) run(i interface{}) error {
	g.tree.updateView(g, i)
	g.tree.setKeybindings(g)

	grid := tview.NewGrid().AddItem(g.tree, 0, 0, 1, 1, 0, 0, true)
	g.pages.AddAndSwitchToPage("main", grid, true)

	if err := g.app.SetRoot(g.pages, true).Run(); err != nil {
		return err
	}

	return nil
}

func (g *yq) modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

func (g *yq) search() {
	pageName := "search"
	if g.pages.HasPage(pageName) {
		g.pages.ShowPage(pageName)
	} else {
		input := tview.NewInputField().SetFieldBackgroundColor(colorDarkGrey)
		input.SetBorder(true).SetTitle("search").SetTitleAlign(tview.AlignLeft)
		input.SetChangedFunc(func(text string) {
			root := *g.tree.root
			g.tree.SetRoot(&root)
			if text != "" {
				root := g.tree.GetRoot()
				root.SetChildren(g.walk(root.GetChildren(), text))
			}
		})
		input.SetLabel("word").SetLabelWidth(5).SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				g.pages.HidePage(pageName)
			}
		})

		g.pages.AddAndSwitchToPage(pageName, g.modal(input, 0, 3), true).ShowPage("main")
	}
}

func (g *yq) walk(nodes []*tview.TreeNode, text string) []*tview.TreeNode {
	var newNodes []*tview.TreeNode

	for _, child := range nodes {
		if strings.Contains(strings.ToLower(child.GetText()), strings.ToLower(text)) {
			newNodes = append(newNodes, child)
		} else {
			newNodes = append(newNodes, g.walk(child.GetChildren(), text)...)
		}
	}

	return newNodes
}

type tree struct {
	*tview.TreeView
	root *tview.TreeNode
}

func newTree() *tree {
	t := &tree{
		TreeView: tview.NewTreeView(),
	}

	// t.SetBorder(true).SetTitle("yaml tree").SetTitleAlign(tview.AlignLeft)
	return t
}

func (t *tree) updateView(g *yq, i interface{}) {
	g.app.QueueUpdateDraw(func() {

		r := newRootTreeNode(i)
		r.SetChildren(t.addNode(i))
		t.SetRoot(r).SetCurrentNode(r)

		root := *r
		t.root = &root
	})
}

func (t *tree) addNode(node interface{}) []*tview.TreeNode {
	var nodes []*tview.TreeNode

	switch node := node.(type) {
	case []interface{}:
		for i, v := range node {
			id := uuid.Must(uuid.NewV4()).String()
			switch v.(type) {
			case map[string]interface{}:
				objectNode := tview.NewTreeNode("{object}").
					SetChildren(t.addNode(v)).SetReference(Reference{ID: id, YAMLType: Object})
				nodes = append(nodes, objectNode)
			case []interface{}:
				nodeName := fmt.Sprintf("{array}%d", i)
				arrayNode := tview.NewTreeNode(nodeName).
					SetChildren(t.addNode(v)).SetReference(Reference{ID: id, YAMLType: Array})
				nodes = append(nodes, arrayNode)
			default:
				nodes = append(nodes, t.addNode(v)...)
			}
		}

	case map[interface{}]interface{}:
		for k, v := range node {
			newNode := t.newNodeWithLiteral(k).
				SetColor(tcell.NewHexColor(16711764)).
				SetChildren(t.addNode(v))
			r := reflect.ValueOf(v)

			id := uuid.Must(uuid.NewV4()).String()
			if r.Kind() == reflect.Slice {
				newNode.SetReference(Reference{ID: id, YAMLType: Array})
			} else if r.Kind() == reflect.Map {
				newNode.SetReference(Reference{ID: id, YAMLType: Object})
			} else {
				newNode.SetReference(Reference{ID: id, YAMLType: Key})
			}

			nodes = append(nodes, newNode)
		}
	default:
		ref := reflect.ValueOf(node)
		var valueType valueType
		switch ref.Kind() {
		case reflect.Int:
			valueType = Int
		case reflect.Float64:
			valueType = Float
		case reflect.Bool:
			valueType = Boolean
		default:
			if node == nil {
				valueType = Null
			} else {
				valueType = String
			}
		}

		id := uuid.Must(uuid.NewV4()).String()
		nodes = append(nodes, t.newNodeWithLiteral(node).
			SetReference(Reference{ID: id, YAMLType: Value, ValueType: valueType}))
	}
	return nodes
}

func (t *tree) newNodeWithLiteral(i interface{}) *tview.TreeNode {
	if i == nil {
		return tview.NewTreeNode("null")
	}
	return tview.NewTreeNode(fmt.Sprintf("%v", i))
}

func (t *tree) setKeybindings(g *yq) {

	t.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 's':
			current := t.GetCurrentNode()
			current.SetExpanded(!current.IsExpanded())
		case 'S':
			t.GetRoot().ExpandAll()
		case 'X':
			t.collapseValues(t.GetRoot())
		case '/', 'f':
			g.search()
		case ' ':
			current := t.GetCurrentNode()
			current.SetExpanded(!current.IsExpanded())
		case 'q':
			g.app.Stop()
		}

		return event
	})
}

func (t *tree) collapseValues(node *tview.TreeNode) {
	node.Walk(func(node, parent *tview.TreeNode) bool {
		ref := node.GetReference().(Reference)
		if ref.YAMLType == Value {
			pRef := parent.GetReference().(Reference)
			t := pRef.YAMLType
			if t == Key || t == Array {
				parent.SetExpanded(false)
			}
		}
		return true
	})
}

func newRootTreeNode(i interface{}) *tview.TreeNode {
	r := reflect.ValueOf(i)

	var root *tview.TreeNode
	switch r.Kind() {
	case reflect.Map:
		root = tview.NewTreeNode("root").SetReference(Reference{YAMLType: Object})
	case reflect.Slice:
		root = tview.NewTreeNode("{array}").SetReference(Reference{YAMLType: Array})
	default:
		root = tview.NewTreeNode("{value}").SetReference(Reference{YAMLType: Key})
	}
	return root
}

func main() {

	input, err := unmarshalYAML(os.Stdin)
	if err != nil || input == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Stdin = os.NewFile(uintptr(syscall.Stderr), "/dev/tty")

	if err := newYQ().run(input); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
