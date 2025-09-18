package global

import (
	"context"
	"fmt"
	"math/big"

	"regexp"
	"strings"
	"time"

	associated_token_account "github.com/gagliardetto/solana-go/programs/associated-token-account"
	token_program "github.com/gagliardetto/solana-go/programs/token"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"

	"github.com/gagliardetto/solana-go/rpc"
	tgbotapi "github.com/zfesd/telegram-bot-api/v6"
)

func CreateSOLAccountOrWrap(instrs *[]solana.Instruction, owner solana.PublicKey, amountIn *big.Int) {
	var wrapAmountNeeded uint64
	bal := SolATA_Balance.Load()

	if SolATA.Load() {
		targetBalance := amountIn.Uint64()
		wsolAccountBalanceU64 := bal.Uint64()
		if wsolAccountBalanceU64 < targetBalance {
			wrapAmountNeeded = targetBalance - wsolAccountBalanceU64
		}
	} else {
		*instrs = append(*instrs, associated_token_account.NewCreateInstruction(owner, owner, solana.WrappedSol).Build())
		wrapAmountNeeded = amountIn.Uint64()
	}

	solATA, _, _ := solana.FindAssociatedTokenAddress(owner, solana.WrappedSol)

	if wrapAmountNeeded > 0 {
		*instrs = append(*instrs, system.NewTransferInstruction(wrapAmountNeeded, owner, solATA).Build())
		*instrs = append(*instrs, token_program.NewSyncNativeInstruction(solATA).Build())
	}
	SolATA.Store(true)
}

func GetBalanceNoDecimals(hexKey string) *big.Float {
	bal := GetBalance(hexKey)
	if bal == nil {
		return Float0
	}
	return new(big.Float).Quo(bal, Float1Lamp)
}

func KeyToWalletPublic(hexKey string) *solana.PublicKey {
	account, err := solana.WalletFromPrivateKeyBase58(hexKey)
	if err != nil {
		fmt.Println("Failed converting key to pub", err)
		return nil
	}
	key := account.PublicKey()
	return &key
}

func GetBalance(hexKey string) *big.Float {
	ctx, done := context.WithTimeout(context.Background(), 3*time.Second)
	defer done()
	out, err := GetRPCForRequest().GetBalance(
		ctx,
		*KeyToWalletPublic(hexKey),
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return Float0
	}
	return new(big.Float).SetUint64(out.Value)
}

func GetBalanceByPublic(publicKey string) *big.Float {
	ctx, done := context.WithTimeout(context.Background(), 3*time.Second)
	pk, _ := solana.PublicKeyFromBase58(publicKey)
	defer done()
	out, err := GetRPCForRequest().GetBalance(
		ctx,
		pk,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return Float0
	}
	return new(big.Float).SetUint64(out.Value)
}

func GetNewWalletPrivateKeyString() string {
	return solana.NewWallet().PrivateKey.String()
}

func ValidatePrivateKey(input string) bool {
	if len(input) < 87 {
		return false
	}
	_, err := solana.PrivateKeyFromBase58(input)
	return err == nil
}

func ValidateAddress(input string) bool {
	_, err := solana.PublicKeyFromBase58(input)
	if err != nil {
		return false
	}
	return true
}

func GetPubKeyFromAddress(input string) solana.PublicKey {
	pubKey, _ := solana.PublicKeyFromBase58(input)
	return pubKey
}

func GetTokenBalanceNoDecimals(account solana.PublicKey, token solana.PublicKey) string {
	tokenAccount, _, err := solana.FindAssociatedTokenAddress(account, token)
	if err != nil {
		fmt.Println("Failed to find token account", err)
		return "0"
	}
	ctx, exp := context.WithTimeout(context.Background(), 3*time.Second)
	defer exp()
	balance, err := GetRPCForRequest().GetTokenAccountBalance(ctx, tokenAccount, "confirmed")
	if err != nil {
		//if !strings.Contains(err.Error(), "could not find account") {
		//	fmt.Println("Failed to query token balance", err)
		//}
		return "0"
	}
	if balance != nil && balance.Value != nil {
		return balance.Value.UiAmountString
	}
	return "0"
}

func KeyToWalletPublicShort(hexKey string) string {
	return KeyToWalletPublic(hexKey).String()[0:5] + "..." + KeyToWalletPublic(hexKey).String()[38:42]
}

func GetGasStringBasedUnit(input int32) string {
	return fmt.Sprint(float64(input) / F1Lamp)
}

func GetGasFromFloat(input float64) int32 {
	return int32(input * F1Lamp)
}

func ConvertGasToString(input int32) string {
	return fmt.Sprint(float64(input) / F1Lamp)
}

func GetCurrentBlockTime() uint64 {
	blockMutex.RLock()
	defer blockMutex.RUnlock()
	return block.Slot
}

func GetTokenBalance(account solana.PublicKey, token solana.PublicKey) (wDecimals *big.Int, woDecimals *big.Float) {
	tokenAccount, _, err := solana.FindAssociatedTokenAddress(account, token)
	if err != nil {
		fmt.Println("Failed to find token account", err)
		return nil, nil
	}
	ctx, exp := context.WithTimeout(context.Background(), 3*time.Second)
	defer exp()
	balance, err := GetRPCForRequest().GetTokenAccountBalance(ctx, tokenAccount, "confirmed")
	if err != nil {
		//if !strings.Contains(err.Error(), "could not find account") {
		//fmt.Println("Failed to query token balance", err)
		//}
		return nil, nil
	}
	if balance != nil && balance.Value != nil {
		wDecimals, _ = new(big.Int).SetString(balance.Value.Amount, 10)
		woDecimals, _ = new(big.Float).SetString(balance.Value.Amount)
	}
	return wDecimals, woDecimals
}

func ReduceDecimals(input *big.Int, decimals int) *big.Float {
	// 10的decimals次方
	ten := big.NewInt(10)
	decimalsBigint := big.NewInt(int64(decimals))
	power := new(big.Int).Exp(ten, decimalsBigint, nil)
	conv := new(big.Float).Quo(new(big.Float).SetInt(input), new(big.Float).SetInt64(power.Int64()))
	return conv
}

func ParseTokenAddressFromStr(url string) string {
	// Regular expression pattern to match an Ethereum token address
	// The address is expected to start with "0x" and be followed by 40 hexadecimal characters.
	pattern := `^[1-9A-HJ-NP-Za-km-z]{32,44}$`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Find the first match of the pattern in the URL
	match := re.FindString(url)

	return match
}

func FindAddress(update *tgbotapi.Update) (bool, string) {
	addr := ParseTokenAddressFromStr(update.Message.Text)
	if len(addr) > 0 {
		return true, addr
	}
	addr = ParseTokenAddressFromStr(update.Message.Caption)
	if len(addr) > 0 {
		return true, addr
	}
	for _, val := range update.Message.CaptionEntities {
		addr = ParseTokenAddressFromStr(val.URL)
		if len(addr) > 0 {
			return true, addr
		}
	}
	for _, val := range update.Message.Entities {
		addr = ParseTokenAddressFromStr(val.URL)
		if len(addr) > 0 {
			return true, addr
		}
	}
	return false, ""
}

func TrimTrailing(str string) string {
	if !strings.Contains(str, ".") {
		return str
	}
	return strings.TrimRight(strings.TrimRight(str, "0"), ".")
}
