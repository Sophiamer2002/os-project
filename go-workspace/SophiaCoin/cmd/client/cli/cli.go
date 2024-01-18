package cli

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os-project/SophiaCoin/pkg/wallet"
	"strconv"
	"time"

	pri "os-project/SophiaCoin/pkg/primitives"
	pb "os-project/SophiaCoin/pkg/rpc"

	"github.com/charmbracelet/bubbles/key"
	inf "github.com/fzdwx/infinite"
	"github.com/fzdwx/infinite/components/input/text"
	"github.com/fzdwx/infinite/components/selection/confirm"
	"github.com/fzdwx/infinite/components/selection/multiselect"
	"github.com/fzdwx/infinite/components/selection/singleselect"
	"github.com/fzdwx/infinite/components/spinner"
	"github.com/fzdwx/infinite/theme"
	"github.com/go-gota/gota/dataframe"
)

type state int

const (
	STATE_INIT state = iota

	STATE_KEY
	STATE_KEY_GEN
	STATE_KEY_VIEW
	STATE_KEY_PUBSAVE

	STATE_PAY

	STATE_BILL
	STATE_BILL_SAVE
)

const layout = "2006-01-02 15:04:05"

var (
	selectKeymap = singleselect.DefaultSingleKeyMap()
)

type Cli struct {
	wallet  *wallet.Wallet
	state   state
	started bool

	server *pb.BroadcastServiceClient
}

func NewCli(server *pb.BroadcastServiceClient, wallet *wallet.Wallet) *Cli {
	return &Cli{
		wallet:  wallet,
		state:   STATE_INIT,
		started: false,
		server:  server,
	}
}

func (cli *Cli) Start() {
	cli.started = true
	fmt.Println("Welcome to SophiaCoin v0.0.1 CLI!")
	cli.state = STATE_INIT
	inf.NewSpinner(
		spinner.WithPrompt("Loading..."),
		spinner.WithDisableOutputResult(),
	).Display(func(spinner *spinner.Spinner) {
		time.Sleep(time.Millisecond * 100 * 12)
		spinner.Info("Start playing with SophiaCoin! Enjoy!")
	})
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
	cli.main()
}

func (cli *Cli) main() {
	var bill dataframe.DataFrame
	for {
		switch cli.state {
		case STATE_INIT:
			menu := inf.NewSingleSelect(
				[]string{
					"Key Management",
					"Payment",
					"Billings",
					"Exit",
				},
				singleselect.WithDisableFilter(),
				singleselect.WithPageSize(5),
				singleselect.WithKeyBinding(selectKeymap),
			)
			choice, err := menu.Display(
				"Main Menu: Select a service",
			)
			if err != nil {
				exit()
			}
			switch choice {
			case 0:
				cli.state = STATE_KEY
			case 1:
				cli.state = STATE_PAY
			case 2:
				cli.state = STATE_BILL
			case 3:
				exit()
			}
		case STATE_KEY:
			menu := inf.NewSingleSelect(
				[]string{
					"Generate a new key",
					"View your keys",
					"Save a known public key",
					"Back",
				},

				singleselect.WithFocusSymbol("->"),
				singleselect.WithDisableFilter(),
				singleselect.WithPageSize(5),
				singleselect.WithKeyBinding(selectKeymap),
			)

			choice, err := menu.Display(
				"Key Management: Select a service",
			)
			if err != nil {
				panic(err)
			}
			switch choice {
			case 0:
				cli.state = STATE_KEY_GEN
			case 1:
				cli.state = STATE_KEY_VIEW
			case 2:
				cli.state = STATE_KEY_PUBSAVE
			case 3:
				cli.state = STATE_INIT
			}

		case STATE_KEY_GEN:
			name := inf.NewText(
				text.WithPrompt("Enter a name for the key:"),
				text.WithPromptStyle(theme.DefaultTheme.PromptStyle),
				text.WithRequired(),
				text.WithRequiredMsg("Name is required(only letters and numbers)"),
				text.WithFocusSymbol("->"),
			)
			name_str, err := name.Display()
			if err != nil {
				panic(err)
			}
			if err := cli.wallet.NewKey(name_str); err != nil {
				fmt.Printf("Generate New Key Error: %v\n", err)
			} else {
				fmt.Println("Generate New Key Success!")
			}
			cli.state = STATE_INIT

		case STATE_KEY_VIEW:
			self_keys := cli.wallet.GetSelfAddress()
			known_keys := cli.wallet.GetKnownAddress()

			keys := make(map[string][]byte)
			for name, key := range self_keys {
				keys[fmt.Sprintf("%s (self)", name)] = key
			}
			for name, key := range known_keys {
				keys[fmt.Sprintf("%s (known)", name)] = key
			}

			options := make([]string, 0, len(keys))

			for name := range keys {
				options = append(options, name)
			}
			menu := inf.NewMultiSelect(
				options,
				multiselect.WithChoiceTextStyle(theme.DefaultTheme.ChoiceTextStyle),
				multiselect.WithFocusSymbol("->"),
				multiselect.WithPageSize(5),
			)

			choices, err := menu.Display(
				"Key Management: Select keys to view",
			)
			if err != nil {
				panic(err)
			}

			for _, choice := range choices {
				fmt.Printf("Key [%s] Address: 0x%x\n", options[choice], keys[options[choice]])
			}

			cli.state = STATE_INIT

		case STATE_KEY_PUBSAVE:
			name := inf.NewText(
				text.WithPrompt("Save a public key of a known address:"),
				text.WithFocusSymbol("->"),
				text.WithRequired(),
				text.WithRequiredMsg("Name is required(only letters and numbers)"),
				text.WithDefaultValue("Alice (maybe)"),
			)

			b := inf.NewText(
				text.WithPrompt("Enter the public key:"),
				text.WithFocusSymbol("->"),
				text.WithRequired(),
				text.WithRequiredMsg("Public key is required(91 bytes)"),
				text.WithDefaultValue("0x"+"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
			)

			name_str, err := name.Display()
			if err != nil {
				panic(err)
			}
			b_str, err := b.Display()
			if err != nil {
				panic(err)
			}

			b_byte, err := hex.DecodeString(b_str[2:])
			if err != nil {
				fmt.Println("Invalid public key")
			} else {
				err = cli.wallet.NewPubAddress(name_str, b_byte)
				if err != nil {
					fmt.Printf("Save Public Key Error: %v\n", err)
				} else {
					fmt.Println("Save Public Key Success!")
				}
			}

			cli.state = STATE_INIT

		case STATE_PAY:
			cli.pay()
			cli.state = STATE_INIT

		case STATE_BILL:
			cli.get_bill(&bill)
		case STATE_BILL_SAVE:
			input := inf.NewText(
				text.WithPrompt("Enter the file name to save:"),
				text.WithFocusSymbol("->"),
				text.WithRequired(),
				text.WithRequiredMsg("File name is required"),
				text.WithDefaultValue("bill.csv"),
			)

			input_str, err := input.Display()
			if err != nil {
				panic(err)
			}

			file, err := os.Create(input_str)
			if err != nil {
				fmt.Printf("Save Bill Error: %v\n", err)
			} else {
				if err = bill.WriteCSV(file); err != nil {
					fmt.Printf("Save Bill Error: %v\n", err)
				} else {
					fmt.Println("Save Bill Success!")
				}
			}
			cli.state = STATE_INIT

		default:
			panic("Unknown state")
		}
	}
}

func (cli *Cli) pay() {
	keys := cli.wallet.GetSelfAddress()
	options := make([]string, 0, len(keys))
	for name := range keys {
		options = append(options, name)
	}
	menu := inf.NewSingleSelect(
		options,
		singleselect.WithFocusSymbol("->"),
		singleselect.WithDisableFilter(),
		singleselect.WithPageSize(5),
		singleselect.WithKeyBinding(selectKeymap),
	)

	choices, err := menu.Display(
		"Payment: Select keys to pay(Ctrl-C to cancel)",
	)
	if err != nil {
		fmt.Println("Payment canceled")
		return
	}

	balance := *cli.wallet.GetBalance()
	fmt.Printf("Key [%s], Balance: %d(Maybe not synchronized)\n", options[choices], balance[options[choices]])

	amount := inf.NewText(
		text.WithPrompt("Enter the amount:"),
		text.WithFocusSymbol("->"),
		text.WithRequired(),
		text.WithRequiredMsg("Amount is required(only numbers, invalid one to quit)"),
		text.WithDefaultValue("1024"),
	)

	amount_str, err := amount.Display()
	if err != nil {
		fmt.Println("Payment canceled")
	}

	amount_int, err := strconv.Atoi(amount_str)
	if err != nil {
		fmt.Println("Payment canceled")
		return
	}

	fee := inf.NewText(
		text.WithPrompt("Enter the fee:"),
		text.WithFocusSymbol("->"),
		text.WithRequired(),
		text.WithRequiredMsg("Fee is required(only numbers, invalid one to quit)"),
		text.WithDefaultValue("1"),
	)

	fee_str, err := fee.Display()
	if err != nil {
		fmt.Println("Payment canceled")
		return
	}

	fee_int, err := strconv.Atoi(fee_str)
	if err != nil {
		fmt.Println("Payment canceled")
		return
	}

	options_ := make([]string, 0)
	pubs := cli.wallet.GetPubAddress()
	for name := range pubs {
		options_ = append(options_, name)
	}
	options_ = append(options_, "Not in the list")

	addr := inf.NewSingleSelect(
		options_,
		singleselect.WithFocusSymbol("->"),
		singleselect.WithDisableFilter(),
		singleselect.WithPageSize(5),
		singleselect.WithKeyBinding(selectKeymap),
	)

	choicepub, err := addr.Display(
		"Payment: Select a destination address(Ctrl-C to cancel)",
	)

	if err != nil {
		fmt.Println("Payment canceled")
		return
	}

	var address []byte
	if choicepub == len(options_)-1 {
		b := inf.NewText(
			text.WithPrompt("Enter the public key to send:"),
			text.WithFocusSymbol("->"),
			text.WithRequired(),
			text.WithRequiredMsg("Public key is required(91 bytes, invalid one to quit)"),
			text.WithDefaultValue("0x"+"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		)
		bb, err := b.Display()
		if err != nil {
			fmt.Println("Payment canceled")
			return
		}

		if len(bb) != 93 || bb[:2] != "0x" {
			fmt.Println("Payment canceled")
			return
		}

		address, err = hex.DecodeString(bb[2:])
		if err != nil {
			fmt.Println("Payment canceled")
			return
		}
	} else {
		address = pubs[options_[choicepub]]
	}

	tx, err := (*(cli.server)).ConstructTransaction(
		context.Background(),
		&pb.TransactionConstruct{
			SendAddr: keys[options[choices]],
			RecvAddr: address,
			Amount:   uint64(amount_int),
			Fee:      uint64(fee_int),
		},
	)

	if err != nil {
		fmt.Printf("Construct Transaction Error: %v\n", err)
		return
	}

	tx_, err := pri.Deserialize(tx.Transaction)

	if err != nil {
		fmt.Printf("Construct Transaction Error: %v\n", err)
		return
	}

	tx__, ok := tx_.(*pri.Transaction)
	if !ok {
		fmt.Printf("Construct Transaction Error\n")
	}

	val, err := inf.NewConfirmWithSelection(
		confirm.WithPrompt("Are you sure to sign this transaction and broadcast it?"),
	).Display()

	if err != nil {
		fmt.Println("Transaction canceled")
	}

	if val {
		cli.wallet.SignTransaction(tx__, options[choices])
		tx_bytes, _ := pri.Serialize(tx__)
		(*cli.server).BroadcastTransaction(context.Background(), &pb.Transaction{
			Transaction: tx_bytes,
		})

		fmt.Println("Transaction broadcast, wait some time to see the result")
	} else {
		fmt.Println("Transaction canceled")
	}
	cli.state = STATE_INIT
}

func (cli *Cli) get_bill(bill *dataframe.DataFrame) {
	keys := cli.wallet.GetSelfAddress()
	options := make([]string, 0, len(keys))

	for name := range keys {
		options = append(options, name)
	}

	menu := inf.NewMultiSelect(
		options,
		multiselect.WithChoiceTextStyle(theme.DefaultTheme.ChoiceTextStyle),
		multiselect.WithFocusSymbol("->"),
		multiselect.WithPrompt("Billings: Select wallets to get bill"),
		multiselect.WithPageSize(5),
	)

	choices, err := menu.Display(
		"Billings: Select wallets to get bill",
	)

	addr := make([]string, 0, len(choices))
	for _, choice := range choices {
		addr = append(addr, options[choice])
	}

	if err != nil {
		panic(err)
	}

	start_time := inf.NewText(
		text.WithPrompt("Enter the start time:"),
		text.WithFocusSymbol("->"),
		text.WithDefaultValue("2000-01-01 00:00:00"),
	)

	end_time := inf.NewText(
		text.WithPrompt("Enter the end time:"),
		text.WithFocusSymbol("->"),
		text.WithDefaultValue("2100-01-01 00:00:00"),
	)

	start_time_str, err := start_time.Display()
	if err != nil {
		panic(err)
	}

	start_time_time, err := time.Parse(layout, start_time_str)
	if err != nil {
		fmt.Println("Invalid start time")
		cli.state = STATE_INIT
		return
	}

	end_time_str, err := end_time.Display()
	if err != nil {
		panic(err)
	}

	end_time_time, err := time.Parse(layout, end_time_str)
	if err != nil {
		fmt.Println("Invalid end time")
		cli.state = STATE_INIT
		return
	}

	*bill = *cli.wallet.GetBill(start_time_time, end_time_time, addr)

	save_or_not := inf.NewConfirmWithSelection(
		confirm.WithPrompt("Do you want to save the bill?"),
	)

	val, err := save_or_not.Display()
	if err != nil {
		panic(err)
	}

	if val {
		cli.state = STATE_BILL_SAVE
	} else {
		cli.state = STATE_INIT

		fmt.Println("Showing the bill:")
		fmt.Println(bill)
	}
}

func exit() {
	inf.NewSpinner(
		spinner.WithPrompt("Exiting..."),
		spinner.WithDisableOutputResult(),
	).Display(func(spinner *spinner.Spinner) {
		time.Sleep(time.Millisecond * 100 * 12)
		spinner.Info("Bye!")
	})
	os.Exit(0)
}
