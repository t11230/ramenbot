package main

import (
    "bytes"
    "fmt"
    "text/tabwriter"
    "math/rand"
    "github.com/bwmarrin/discordgo"
    "strconv"
    "strings"
    "time"
    log "github.com/Sirupsen/logrus"
)

type Player struct {
    UserID string
    Bid int
}

type BetRoll struct {
    Players []Player
    Ante int
}

func rollDice(guild *discordgo.Guild, message *discordgo.Message, args []string) string{
    roll_help := `**roll usage:** roll *dietype (optional)*
    This command initiates a dice roll.
    The second optional argument specifies a type of die for the roll.
    **Die Types**
    **d6 (default):** 6-sided die.
    **d4:** 4-sided die.
    **d8:** 8-sided die.
    **d10:** 10-sided die.
    **d12:** 12-sided die.
    **d20:** 20-sided die.
    **other:** random integer generator between 1 and input.`
    draw := false
    r:=0
    maxnum:=0
    w := &tabwriter.Writer{}
    buf := &bytes.Buffer{}
    w.Init(buf, 0, 4, 0, ' ', 0)
    var err error
    if len(args)>1 {
        if (len(args)>2) || (args[1] == "help") {
            fmt.Fprintf(w, roll_help)
            w.Flush()
            return buf.String()
            }
        if strings.HasPrefix(args[1], "d") {
            maxnum, err = strconv.Atoi(strings.Replace(args[1], "d", "", 1))
            if err!=nil {
                return ""
            }
            if isValidDie(maxnum) {
                draw = true
            }
        } else {
            maxnum, err = strconv.Atoi(args[1])
            if err!=nil {
                return ""
            }
        }
        r = rand.Intn(maxnum) + 1
    } else {
        maxnum = 6
        r = rand.Intn(6) + 1
        draw = true
    }
    result := ""
    if draw {
        if maxnum == 6 {
            result = drawD6(r)
        } else if (maxnum == 4) || (maxnum == 8) {
            result = drawD4_D8(r)
        } else if maxnum == 10 {
            result = drawD10(r)
        } else if maxnum == 12 {
            result = drawD12(r)
        } else if maxnum == 20 {
            result = drawD20(r)
        }
    } else{
        result = "The result is: "+strconv.Itoa(r)
    }
    fmt.Fprintf(w, "```\n")
    fmt.Fprintf(w, result)
    fmt.Fprintf(w, "```\n")
    w.Flush()
    return buf.String()
}

func betRoll(guild *discordgo.Guild, message *discordgo.Message, args []string) string {
    log.Info("Placing Betroll")
    db := dbGetSession(guild.ID)
    event := db.GetActiveBetRoll(guild.ID)
    pool := 0
    ante := 0
    payout := 0
    draw := false
    r:=0
    maxnum:=0
    w := &tabwriter.Writer{}
    buf := &bytes.Buffer{}
    result := ""
    win_result := "Winner(s):\n"
    payout_result := "Payout: "
    w.Init(buf, 0, 4, 0, ' ', 0)
    var err error
    betroll_help := `**betroll usage:** betroll *ante* *dietype (optional)*
    This command initiates a bet on a dice roll. The second argument is the ante that all participants must pay into the pool.
    The third optional argument specifies a type of die for the roll.
    **Die Types**
    **d6 (default):** 6-sided die.
    **d4:** 4-sided die.
    **d8:** 8-sided die.
    **d10:** 10-sided die.
    **d12:** 12-sided die.
    **d20:** 20-sided die.
    **other:** random integer generator between 1 and input.`
    ante_error_msg := "**ERROR:** Non-numerical ante submitted.  Please don't be a smartass."
    dice_error_msg := "**ERROR:** Non-numerical dice submitted.  Please don't be a smartass."
    event_error_msg := "**ERROR:** BetRoll Event already in progress."
    db_error_msg := "**ERROR:** Database error."
    //success_msg := ""
    var players []Player
    var winnerIDs []string
    if event != nil {
        fmt.Fprintf(w, event_error_msg)
        w.Flush()
        return buf.String()
    }
    err = db.BetRollOpen(guild.ID)
    if err != nil {
        fmt.Fprintf(w, db_error_msg)
        w.Flush()
        return buf.String()
    }
    if (len(args)<2) || (len(args)>3) {
        fmt.Fprintf(w, betroll_help)
        w.Flush()
        return buf.String()
    } else if len(args)>2 {
        ante, err = strconv.Atoi(args[1])
        if err != nil {
            fmt.Fprintf(w, ante_error_msg)
            w.Flush()
            return buf.String()
        }
        log.Info(ante)
        err = db.SetBetRollAnte(guild.ID, ante)
        if err != nil {
            fmt.Fprintf(w, db_error_msg)
            w.Flush()
            return buf.String()
        }
        printBetRollTime(30, ante)
        time.Sleep(10*time.Second)
        printBetRollTime(20, ante)
        time.Sleep(10*time.Second)
        printBetRollTime(10, ante)
        time.Sleep(10*time.Second)
        printBetRollTime(0, ante)
        players = db.GetPlayers(guild.ID)
        pool = ante * len(players)
        if strings.HasPrefix(args[2], "d") {
            maxnum, err = strconv.Atoi(strings.Replace(args[2], "d", "", 1))
            if err!=nil {
                fmt.Fprintf(w, dice_error_msg)
                w.Flush()
                return buf.String()
            }
            if isValidDie(maxnum) {
                draw = true
            }
        } else {
            if err!=nil {
                fmt.Fprintf(w, dice_error_msg)
                w.Flush()
                return buf.String()
            }
        }
        r = rand.Intn(maxnum) + 1
        if draw {
            if maxnum == 6 {
                result = drawD6(r)
            } else if (maxnum == 4) || (maxnum == 8) {
                result = drawD4_D8(r)
            } else if maxnum == 10 {
                result = drawD10(r)
            } else if maxnum == 12 {
                result = drawD12(r)
            } else if maxnum == 20 {
                result = drawD20(r)
            }
        } else{
            result = "The result is: "+strconv.Itoa(r)
        }
        for _,player := range(players) {
            if player.Bid == r {
                winnerIDs = append(winnerIDs, player.UserID)
            }
        }
        if len(winnerIDs) == 0 {
            payout = 0
        } else {
            payout = pool/len(winnerIDs)
        }
        payout_result = payout_result + strconv.Itoa(payout) + " bits"
        for _,winner := range(winnerIDs) {
            db.IncBitStats(winner, payout)
            win_result = win_result + utilGetPreferredName(guild, winner) + "\n"
        }
    }else {
        ante, err = strconv.Atoi(args[1])
        if err != nil {
            fmt.Fprintf(w, ante_error_msg)
            w.Flush()
            return buf.String()
        }
        err = db.SetBetRollAnte(guild.ID, ante)
        if err != nil {
            fmt.Fprintf(w, db_error_msg)
            w.Flush()
            return buf.String()
        }
        printBetRollTime(30, ante)
        time.Sleep(10*time.Second)
        printBetRollTime(20, ante)
        time.Sleep(10*time.Second)
        printBetRollTime(10, ante)
        time.Sleep(10*time.Second)
        printBetRollTime(0, ante)
        players = db.GetPlayers(guild.ID)
        pool = ante * len(players)
        maxnum = 6
        r = rand.Intn(6) + 1
        result = drawD6(r)
        for _,player := range(players) {
            if player.Bid == r {
                winnerIDs = append(winnerIDs, player.UserID)
            }
        }
        if len(winnerIDs) == 0 {
            payout = 0
        } else {
            payout = pool/len(winnerIDs)
        }
        payout_result = payout_result + strconv.Itoa(payout) + " bits"
        for _,winner := range(winnerIDs) {
            db.IncBitStats(winner, payout)
            win_result = win_result + "     " + utilGetPreferredName(guild, winner) + "\n"
        }
    }
    err = db.BetRollClose(guild.ID)
    if err!=nil {
        fmt.Fprintf(w, db_error_msg)
        w.Flush()
        return buf.String()
    }

    fmt.Fprintf(w, "```\n")
    fmt.Fprintf(w, result)
    fmt.Fprintf(w, "```\n")
    fmt.Fprintf(w, "```\n")
    fmt.Fprintf(w, win_result)
    fmt.Fprintf(w, payout_result)
    fmt.Fprintf(w, "```\n")
    w.Flush()
    return buf.String()
}

func bid(guild *discordgo.Guild, message *discordgo.Message, args []string) string {
    user := message.Author
    db := dbGetSession(guild.ID)
    event := db.GetActiveBetRoll(guild.ID)
    w := &tabwriter.Writer{}
    buf := &bytes.Buffer{}
    var err error
    w.Init(buf, 0, 4, 0, ' ', 0)
    event_error_msg := "**ERROR:** No BetRoll Event currently in progress."
    bid_error_msg := "**ERROR:** Non-numerical bid submitted.  Please don't be a smartass."
    db_error_msg := "**ERROR:** Database error."
    success_msg := "*Bet Successfully Placed* :ok_hand:"
    ante_error_msg := "Not enough bits for ante :slight_frown:"
    bid_help := `**bid usage:** bid *number*
    This command bids on a bet roll. The second argument is the result that you are bidding on.`
    if event == nil {
        fmt.Fprintf(w, event_error_msg)
        w.Flush()
        return buf.String()
    }
    if len(args)!=2 {
        fmt.Fprintf(w, bid_help)
        w.Flush()
        return buf.String()
    }
    var me Player
    me.UserID = user.ID
    me.Bid, err = strconv.Atoi(args[1])
    if err!=nil {
        fmt.Fprintf(w, bid_error_msg)
        w.Flush()
        return buf.String()
    }
    err = db.DecCheckBitStats(user.ID, event.Ante)
    if err!=nil {
        fmt.Fprintf(w, ante_error_msg)
        w.Flush()
        return buf.String()
    }
    db.DecBitStats(user.ID, event.Ante)
    err = db.BetRollAddPlayer(guild.ID, me)
    if err!=nil {
        fmt.Fprintf(w, db_error_msg)
        w.Flush()
        return buf.String()
    }
    fmt.Fprintf(w, success_msg)
    w.Flush()
    return buf.String()
}

func isValidDie(num int) bool {
    return utilIntInSlice(num, []int{4,6,8,10,12,20})
}

func drawD6(r int) string {
    C := "o "
    s:="---------\n| "+string(C[utilBooltoInt(r<=1)])+"   "+string(C[utilBooltoInt(r<=3)])+" |\n| "+string(C[utilBooltoInt(r<=5)])
    z:=string(C[utilBooltoInt(r<=5)])+" |\n| "+string(C[utilBooltoInt(r<=3)])+"   "+string(C[utilBooltoInt(r<=1)])+" |\n---------"
    return s+" "+string(C[utilBooltoInt((r&1)==0)])+" "+z
}

func drawD4_D8(r int) string {
    return "      *\n     * *\n    *   *\n   *  "+strconv.Itoa(r)+"  *\n  *       *\n * * * * * *"
}

func drawD10(r int) string {
    numstring := strconv.Itoa(r)
    if r > 9 {
        return "        *\n       * *\n      *   *\n     * "+string(numstring[0])+" "+string(numstring[1])+" *\n      *   *\n        *"
    } else {
        return "        *\n       * *\n      *   *\n     *  "+numstring+"  *\n      *   *\n        *"
    }
}

func drawD12(r int) string {
    numstring := strconv.Itoa(r)
    if r > 9 {
        return "         *\n      *     *\n    *   "+string(numstring[0])+" "+string(numstring[1])+"   *\n     *       *\n      * * * *"
    } else {
        return "         *\n      *     *\n    *    "+numstring+"    *\n     *       *\n      * * * *"
    }
}

func drawD20(r int) string {
    numstring := strconv.Itoa(r)
    if r > 9 {
        return "      *\n     * *\n    *   *\n   * "+string(numstring[0])+" "+string(numstring[1])+" *\n  *       *\n * * * * * *"
    } else {
        return "      *\n     * *\n    *   *\n   *  "+numstring+"  *\n  *       *\n * * * * * *"
    }
}

func printBetRollTime(time int, ante int) string {
    w := &tabwriter.Writer{}
    buf := &bytes.Buffer{}
    alert:= ""
    w.Init(buf, 0, 4, 0, ' ', 0)
    if time == 30 {
        alert = "Dice Roll in **"+strconv.Itoa(time)+" seconds**.  Ante is **"+strconv.Itoa(ante)+" bits**. !!bid *result* to bid"
    } else if time != 0 {
        alert = "Dice Roll in **"+strconv.Itoa(time)+" seconds**."
    } else {
        alert = "**Dice Roll starting now!**"
    }
    fmt.Fprintf(w, alert)
    w.Flush()
    return buf.String()
}
