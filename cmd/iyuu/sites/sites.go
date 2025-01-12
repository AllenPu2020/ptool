package sites

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sagan/ptool/cmd/iyuu"
	"github.com/sagan/ptool/config"
	"github.com/sagan/ptool/utils"
)

var command = &cobra.Command{
	Use:   "sites",
	Short: "Show iyuu sites list",
	Long:  `Show iyuu sites list`,
	Run:   sites,
}

var (
	showBindable = false
	showAll      = false
	filter       = ""
)

func init() {
	command.Flags().BoolVarP(&showBindable, "bindable", "b", false, "Show bindable sites")
	command.Flags().StringVarP(&filter, "filter", "f", "", "Filter sites. Only show sites which name / url / comment contain this string")
	command.Flags().BoolVarP(&showAll, "all", "a", false, "Show all iyuu sites (instead of only owned sites)")
	iyuu.Command.AddCommand(command)
}

func sites(cmd *cobra.Command, args []string) {
	if showBindable {
		bindableSites, err := iyuu.IyuuApiGetRecommendSites()
		if err != nil {
			log.Fatalf("Failed to get iyuu bindable sites: %v", err)
		}
		fmt.Printf("%-20s  %7s  %20s\n", "SiteName", "SiteId", "BindParams")
		for _, site := range bindableSites {
			if filter != "" && (!utils.ContainsI(site.Site, filter) && fmt.Sprint(site.Id) != filter) {
				continue
			}
			fmt.Printf("%-20s  %7d  %20s\n", site.Site, site.Id, site.Bind_check)
		}
		os.Exit(0)
	}

	log.Tracef("iyuu token: %s", config.Get().IyuuToken)
	if config.Get().IyuuToken == "" {
		log.Fatalf("You must config iyuuToken in ptool.toml to use iyuu functions")
	}

	iyuuApiSites, err := iyuu.IyuuApiSites(config.Get().IyuuToken)
	if err != nil {
		log.Fatalf("Failed to get iyuu sites: %v", err)
	}
	iyuuSites := utils.Map(iyuuApiSites, func(site iyuu.IyuuApiSite) iyuu.Site {
		return site.ToSite()
	})
	log.Printf("Iyuu sites: len(sites)=%v\n", len(iyuuSites))
	iyuu2LocalSiteMap := iyuu.GenerateIyuu2LocalSiteMap(iyuuSites, config.Get().Sites)

	if showAll {
		fmt.Printf("<all iyuu supported sites>\n")
	} else {
		fmt.Printf("<local sites supported by iyuu> (add -a flag to show all iyuu sites)\n")
	}
	fmt.Printf("%-10s  %-15s  %-6s  %-13s  %-30s  %-25s\n", "Nickname", "SiteName", "SiteId", "LocalSite", "SiteUrl", "DlPage")
	for _, iyuuSite := range iyuuSites {
		if iyuu2LocalSiteMap[iyuuSite.Sid] == "" {
			continue
		}
		if filter != "" && !matchFilter(&iyuuSite, filter) {
			continue
		}
		utils.PrintStringInWidth(iyuuSite.Nickname, 10, true)
		fmt.Printf("  %-15s  %-6d  %-13s  %-30s  %-25s\n", iyuuSite.Name, iyuuSite.Sid,
			iyuu2LocalSiteMap[iyuuSite.Sid], iyuuSite.Url, iyuuSite.DownloadPage)
	}

	if showAll {
		for _, iyuuSite := range iyuuSites {
			if iyuu2LocalSiteMap[iyuuSite.Sid] != "" {
				continue
			}
			if filter != "" && !matchFilter(&iyuuSite, filter) {
				continue
			}
			utils.PrintStringInWidth(iyuuSite.Nickname, 10, true)
			fmt.Printf("  %-15s  %-6d  %-13s  %-30s  %-25s\n", iyuuSite.Name, iyuuSite.Sid,
				"X (None)", iyuuSite.Url, iyuuSite.DownloadPage)
		}
	}
}

func matchFilter(iyuuSite *iyuu.Site, filter string) bool {
	return utils.ContainsI(iyuuSite.Name, filter) ||
		utils.ContainsI(iyuuSite.Nickname, filter) ||
		utils.ContainsI(iyuuSite.Url, filter) ||
		fmt.Sprint(iyuuSite.Sid) == filter
}
