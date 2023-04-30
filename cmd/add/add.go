package add

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	goTorrentParser "github.com/j-muller/go-torrent-parser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sagan/ptool/client"
	"github.com/sagan/ptool/cmd"
	"github.com/sagan/ptool/site"
	"github.com/sagan/ptool/utils"
)

var command = &cobra.Command{
	Use:   "add <client> <torrentIdOrUrl>...",
	Short: "Add site torrents to client",
	Long:  `Add site torrents to client.`,
	Args:  cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
	Run:   add,
}

var (
	paused      = false
	addCategory = ""
	defaultSite = ""
	addTags     = ""
)

func init() {
	command.Flags().BoolVarP(&paused, "paused", "p", false, "Add torrents to client in paused state")
	command.Flags().StringVar(&addCategory, "add-category", "", "Set category of added torrents.")
	command.Flags().StringVar(&defaultSite, "site", "", "Set default site of torrents")
	command.Flags().StringVar(&addTags, "add-tags", "", "Add tags to added torrent (comma-separated).")
	cmd.RootCmd.AddCommand(command)
}

func add(cmd *cobra.Command, args []string) {
	clientInstance, err := client.CreateClient(args[0])
	if err != nil {
		log.Fatal(err)
	}

	siteInstanceMap := make(map[string](site.Site))
	errCnt := int64(0)
	torrentIds := args[1:]
	option := &client.TorrentOption{
		Pause:    paused,
		Category: addCategory,
	}
	if addTags != "" {
		option.Tags = strings.Split(addTags, ",")
	}

	for _, torrentId := range torrentIds {
		siteName := defaultSite
		if !utils.IsUrl(torrentId) {
			i := strings.Index(torrentId, ".")
			if i != -1 && i < len(torrentId)-1 {
				siteName = torrentId[:i]
				torrentId = torrentId[i+1:]
			}
		}
		if siteName == "" {
			fmt.Printf("torrent %s: no site provided", torrentId)
			errCnt++
			continue
		}
		if siteInstanceMap[siteName] == nil {
			siteInstance, err := site.CreateSite(siteName)
			if err != nil {
				log.Fatalf("Failed to create site %s: %v", siteName, err)
			}
			siteInstanceMap[siteName] = siteInstance
		}
		siteInstance := siteInstanceMap[siteName]
		torrentContent, _, err := siteInstance.DownloadTorrent(torrentId)
		if err != nil {
			fmt.Printf("add site %s torrent %s error: failed to get site torrent: %v\n", siteInstance.GetName(), torrentId, err)
			errCnt++
			continue
		}
		tinfo, err := goTorrentParser.Parse(bytes.NewReader(torrentContent))
		if err != nil {
			fmt.Printf("add site %s torrent %s error: failed to parse torrent: %v\n", siteInstance.GetName(), torrentId, err)
			errCnt++
			continue
		}
		err = clientInstance.AddTorrent(torrentContent, option, nil)
		if err != nil {
			fmt.Printf("add site %s torrent %s error: failed to add torrent to client: %v\n", siteInstance.GetName(), torrentId, err)
			errCnt++
			continue
		}
		fmt.Printf("add site %s torrent %s success. infoHash=%s\n", siteInstance.GetName(), torrentId, tinfo.InfoHash)
	}
	if errCnt > 0 {
		os.Exit(1)
	}
}
