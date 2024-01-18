package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	inf "github.com/fzdwx/infinite"
	"github.com/fzdwx/infinite/components"
	"github.com/fzdwx/infinite/components/input/text"
	"github.com/fzdwx/infinite/components/selection/confirm"
	"github.com/fzdwx/infinite/components/selection/singleselect"
	"github.com/fzdwx/infinite/theme"
	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

func main() {
	csvStr := `
Country,Date,Age,Amount,Id
"United States",2012-02-01,50,112.1,01234
"United States",2012-02-01,32,321.31,54320
"United Kingdom",2012-02-01,17,18.2,12345
"United States",2012-02-01,32,321.31,54320
"United Kingdom",2012-02-01,NA,18.2,12345
"United States",2012-02-01,32,321.31,54320
"United States",2012-02-01,32,321.31,54320
Spain,2012-02-01,66,555.42,00241
`
	df := dataframe.ReadCSV(strings.NewReader(csvStr))
	fmt.Println(df)
	fmt.Printf("df[0][0] = (%T)%v\n", df.Elem(0, 0), df.Elem(0, 0))
	df.Elem(0, 0)
	fil := df.Filter(
		dataframe.F{
			Colname:    "Amount",
			Comparator: series.Less,
			Comparando: 321.31,
		},
	)
	fmt.Println(fil)

	df_ := df.GroupBy("Country").Aggregation(
		[]dataframe.AggregationType{dataframe.Aggregation_SUM, dataframe.Aggregation_SUM},
		[]string{"Amount", "Id"},
	)
	fmt.Println(df_)

	df1 := dataframe.LoadRecords(
		[][]string{
			{"A", "B", "C", "D"},
			{"a", "4", "5.1", "true"},
			{"k", "5", "7.0", "true"},
			{"k", "4", "6.0", "true"},
			{"a", "2", "7.1", "false"},
		},
	)
	df2 := dataframe.LoadRecords(
		[][]string{
			{"A", "F", "D"},
			{"1", "1", "true"},
			{"4", "2", "false"},
			{"2", "8", "false"},
			{"5", "9", "false"},
		},
	)
	join := df1.InnerJoin(df2, "D")
	fmt.Println(join)

	df = dataframe.LoadRecords(
		[][]string{
			{"A", "B", "C", "D"},
			{"a", "4", "5.1", "true"},
			{"b", "4", "6.0", "true"},
			{"c", "3", "6.0", "false"},
			{"a", "2", "7.1", "false"},
		},
	)
	fmt.Println(df)

	type User struct {
		Name     string
		Age      int
		Accuracy float64
		ignored  bool // ignored since unexported
	}
	users := []User{
		{"Aram", 17, 0.2, true},
		{"Juan", 18, 0.8, true},
		{"Ana", 22, 0.5, true},
	}
	df = dataframe.LoadStructs([]User{users[0]})
	fmt.Println(df)

	df = dataframe.New(
		series.New([]string{}, series.String, "Name"),
		series.New([]int{}, series.Int, "Age"),
		series.New([]float64{}, series.Float, "Accuracy"),
	)
	df = df.RBind(dataframe.LoadStructs(users))
	fmt.Println(df)

	ele := df.Elem(0, 0).Float()
	fmt.Println("df.Elem(0, 1).Int() =", ele)

	val, _ := inf.NewConfirmWithSelection(
		confirm.WithPrompt("are you a human?"),
		// confirm.WithDefaultYes(),
	).Display()

	if val {
		fmt.Println("yes, you are.")
	} else {
		fmt.Println("no,you are not.")
	}

	input := components.NewInput()

	if err := components.NewStartUp(input).Start(); err != nil {
		panic(err)
	}

	fmt.Println("input:", input.Value())

	i := inf.NewText(
		text.WithPrompt("what's your name?"),
		text.WithPromptStyle(theme.DefaultTheme.PromptStyle),
		text.WithDefaultValue("fzdwx (maybe)"),
	)

	vall, err := i.Display()

	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	fmt.Printf("you input: %s\n", vall)

	options := []string{
		"1 Buy carrots",
		"2 Buy celery",
		"3 Buy kohlrabi",
		"4 Buy computer",
		"5 Buy something",
		"6 Buy car",
		"7 Buy subway",
	}

	selectKeymap := singleselect.DefaultSingleKeyMap()
	selectKeymap.Confirm = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "finish select"),
	)
	selectKeymap.Choice = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "finish select"),
	)
	selectKeymap.NextPage = key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("->", "next page"),
	)
	selectKeymap.PrevPage = key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("<-", "prev page"),
	)
	selected, err := inf.NewSingleSelect(
		options,
		singleselect.WithDisableFilter(),
		singleselect.WithKeyBinding(selectKeymap),
		singleselect.WithPageSize(5),
	).Display("Hello world")

	if err == nil {
		fmt.Printf("you selection %s\n", options[selected])
	}
}
