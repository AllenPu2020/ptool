package qbittorrent

// qb web API: https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"github.com/sagan/ptool/client"
	"github.com/sagan/ptool/config"
	"github.com/sagan/ptool/utils"
)

type Client struct {
	Name         string
	ClientConfig *config.ClientConfigStruct
	Config       *config.ConfigStruct
	HttpClient   *http.Client
	data         *apiSyncMaindata
	Logined      bool
	NoLogin      bool
	datatime     int64
}

func (qbclient *Client) apiPost(apiUrl string, data url.Values) error {
	resp, err := qbclient.HttpClient.PostForm(qbclient.ClientConfig.Url+apiUrl, data)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) == "Fails." {
		return fmt.Errorf("apiPost error: Fails")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("apiPost error: status=%d", resp.StatusCode)
	}
	return nil
}

func (qbclient *Client) apiRequest(apiPath string, v any) error {
	resp, err := qbclient.HttpClient.Get(qbclient.ClientConfig.Url + apiPath)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("apiRequest %s response %d status", apiPath, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if v != nil {
		return json.Unmarshal(body, &v)
	} else {
		return nil
	}
}

func (qbclient *Client) login() error {
	if qbclient.Logined || qbclient.NoLogin {
		return nil
	}
	username := qbclient.ClientConfig.Username
	password := qbclient.ClientConfig.Password
	// use qb default
	if username == "" {
		username = "admin"
	}
	if username == "admin" && password == "" {
		password = "adminadmin"
	}
	data := url.Values{
		"username": {username},
		"password": {password},
	}
	err := qbclient.apiPost("api/v2/auth/login", data)
	if err == nil {
		qbclient.Logined = true
	}
	return err
}

func (qbclient *Client) GetName() string {
	return qbclient.Name
}

func (qbclient *Client) GetClientConfig() *config.ClientConfigStruct {
	return qbclient.ClientConfig
}

func (qbclient *Client) AddTorrent(torrentContent []byte, option *client.TorrentOption, meta map[string](int64)) error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}

	name := client.GenerateNameWithMeta(option.Name, meta)
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()
	// see https://stackoverflow.com/questions/21130566/how-to-set-content-type-for-a-form-filed-using-multipart-in-go
	// torrentPartWriter, _ := mp.CreateFormField("torrents")
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="torrents"; filename="file.torrent"`)
	h.Set("Content-Type", "application/x-bittorrent")
	torrentPartWriter, err := mp.CreatePart(h)
	if err != nil {
		return err
	}
	torrentPartWriter.Write(torrentContent)
	mp.WriteField("rename", name)
	mp.WriteField("root_folder", "true")
	if option != nil {
		mp.WriteField("category", option.Category)
		mp.WriteField("tags", strings.Join(option.Tags, ",")) // qb 4.3.2+ new
		mp.WriteField("paused", fmt.Sprint(option.Pause))
		mp.WriteField("upLimit", fmt.Sprint(option.UploadSpeedLimit))
		mp.WriteField("dlLimit", fmt.Sprint(option.DownloadSpeedLimit))
		if option.SavePath != "" {
			mp.WriteField("savepath", option.SavePath)
			mp.WriteField("autoTMM", "false")
		}
		if option.SkipChecking {
			mp.WriteField("skip_checking", "true")
		}
	}
	resp, err := qbclient.HttpClient.Post(qbclient.ClientConfig.Url+"api/v2/torrents/add",
		mp.FormDataContentType(), body)
	if err != nil {
		return fmt.Errorf("add torrent error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("add torrent error: status=%d", resp.StatusCode)
	}
	return err
}

func (qbclient *Client) PauseTorrents(infoHashes []string) error {
	if len(infoHashes) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {strings.Join(infoHashes, "|")},
	}
	return qbclient.apiPost("api/v2/torrents/pause", data)
}

func (qbclient *Client) ResumeTorrents(infoHashes []string) error {
	if len(infoHashes) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {strings.Join(infoHashes, "|")},
	}
	return qbclient.apiPost("api/v2/torrents/resume", data)
}

func (qbclient *Client) RecheckTorrents(infoHashes []string) error {
	if len(infoHashes) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {strings.Join(infoHashes, "|")},
	}
	return qbclient.apiPost("api/v2/torrents/recheck", data)
}

func (qbclient *Client) ReannounceTorrents(infoHashes []string) error {
	if len(infoHashes) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {strings.Join(infoHashes, "|")},
	}
	return qbclient.apiPost("api/v2/torrents/reannounce", data)
}

func (qbclient *Client) AddTagsToTorrents(infoHashes []string, tags []string) error {
	if len(infoHashes) == 0 || len(tags) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {strings.Join(infoHashes, "|")},
		"tags":   {strings.Join(tags, ",")},
	}
	return qbclient.apiPost("api/v2/torrents/addTags", data)
}

func (qbclient *Client) RemoveTagsFromTorrents(infoHashes []string, tags []string) error {
	if len(infoHashes) == 0 || len(tags) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {strings.Join(infoHashes, "|")},
		"tags":   {strings.Join(tags, ",")},
	}
	return qbclient.apiPost("api/v2/torrents/removeTags", data)
}

func (qbclient *Client) SetTorrentsSavePath(infoHashes []string, savePath string) error {
	if len(infoHashes) == 0 {
		return nil
	}
	savePath = strings.TrimSpace(savePath)
	if savePath == "" {
		return fmt.Errorf("savePath is empty")
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes":   {strings.Join(infoHashes, "|")},
		"location": {savePath},
	}
	return qbclient.apiPost("api/v2/torrents/setLocation", data)
}

func (qbclient *Client) PauseAllTorrents() error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {"all"},
	}
	return qbclient.apiPost("api/v2/torrents/pause", data)
}

func (qbclient *Client) ResumeAllTorrents() error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {"all"},
	}
	return qbclient.apiPost("api/v2/torrents/resume", data)
}

func (qbclient *Client) RecheckAllTorrents() error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {"all"},
	}
	return qbclient.apiPost("api/v2/torrents/recheck", data)
}

func (qbclient *Client) ReannounceAllTorrents() error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {"all"},
	}
	return qbclient.apiPost("api/v2/torrents/reannounce", data)
}

func (qbclient *Client) AddTagsToAllTorrents(tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {"all"},
		"tags":   {strings.Join(tags, ",")},
	}
	return qbclient.apiPost("api/v2/torrents/addTags", data)
}

func (qbclient *Client) RemoveTagsFromAllTorrents(tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes": {"all"},
		"tags":   {strings.Join(tags, ",")},
	}
	return qbclient.apiPost("api/v2/torrents/removeTags", data)
}

func (qbclient *Client) SetAllTorrentsSavePath(savePath string) error {
	savePath = strings.TrimSpace(savePath)
	if savePath == "" {
		return fmt.Errorf("savePath is empty")
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes":   {"all"},
		"location": {savePath},
	}
	return qbclient.apiPost("api/v2/torrents/setLocation", data)
}

func (qbclient *Client) GetTags() ([]string, error) {
	err := qbclient.login()
	if err != nil {
		return nil, fmt.Errorf("login error: %v", err)
	}
	var tags []string
	err = qbclient.apiRequest("api/v2/torrents/tags", &tags)
	return tags, err
}

func (qbclient *Client) CreateTags(tags ...string) error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"tags": {strings.Join(tags, ",")},
	}
	return qbclient.apiPost("api/v2/torrents/createTags", data)
}

func (qbclient *Client) DeleteTags(tags ...string) error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"tags": {strings.Join(tags, ",")},
	}
	return qbclient.apiPost("api/v2/torrents/deleteTags", data)
}

func (qbclient *Client) GetCategories() ([]string, error) {
	err := qbclient.login()
	if err != nil {
		return nil, fmt.Errorf("login error: %v", err)
	}
	var categories map[string](apiCategoryStruct)
	err = qbclient.apiRequest("api/v2/torrents/categories", &categories)
	if err != nil {
		return nil, err
	}
	cats := []string{}
	for _, category := range categories {
		cats = append(cats, category.Name)
	}
	return cats, nil
}

func (qbclient *Client) SetTorrentsCatetory(infoHashes []string, category string) error {
	if len(infoHashes) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes":   {strings.Join(infoHashes, "|")},
		"category": {category},
	}
	return qbclient.apiPost("api/v2/torrents/setCategory", data)
}

func (qbclient *Client) SetAllTorrentsCatetory(category string) error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	data := url.Values{
		"hashes":   {"all"},
		"category": {category},
	}
	return qbclient.apiPost("api/v2/torrents/setCategory", data)
}

func (qbclient *Client) DeleteTorrents(infoHashes []string, deleteFiles bool) error {
	if len(infoHashes) == 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}
	data := url.Values{
		"hashes":      {strings.Join(infoHashes, "|")},
		"deleteFiles": {deleteFilesStr},
	}
	return qbclient.apiPost("api/v2/torrents/delete", data)
}

func (qbclient *Client) ModifyTorrent(infoHash string,
	option *client.TorrentOption, meta map[string](int64)) error {
	if option == nil {
		option = &client.TorrentOption{}
	}
	err := qbclient.sync()
	if err != nil {
		return err
	}

	// qbtorrent := &apiTorrentProperties{}
	// err = qbclient.apiRequest("api/v2/torrents/properties?hash="+torrent.InfoHash, qbtorrent)
	qbtorrent, ok := qbclient.data.Torrents[infoHash]
	if !ok {
		return fmt.Errorf("torrent not exists")
	}

	if option.Name != "" || len(meta) > 0 {
		name := option.Name
		if name == "" {
			name, _ = client.ParseMetaFromName(qbtorrent.Name)
		}
		name = client.GenerateNameWithMeta(name, meta)
		if name != qbtorrent.Name {
			data := url.Values{
				"hash": {infoHash},
				"name": {name},
			}
			err := qbclient.apiPost("api/v2/torrents/rename", data)
			if err != nil {
				return err
			}
		}
	}

	if option.Category != "" && option.Category != qbtorrent.Category {
		data := url.Values{
			"hashes":   {infoHash},
			"category": {option.Category},
		}
		err := qbclient.apiPost("api/v2/torrents/setCategory", data)
		if err != nil {
			return err
		}
	}

	if len(option.Tags) > 0 || len(option.RemoveTags) > 0 {
		qbTags := strings.Split(qbtorrent.Tags, ",")
		addTags := []string{}
		removeTags := []string{}
		for _, addTag := range option.Tags {
			if slices.Index(qbTags, addTag) == -1 {
				addTags = append(addTags, addTag)
			}
		}
		for _, removeTag := range option.RemoveTags {
			if slices.Index(qbTags, removeTag) != -1 {
				removeTags = append(removeTags, removeTag)
			}
		}
		if len(removeTags) > 0 {
			data := url.Values{
				"hashes": {infoHash},
				"tags":   {strings.Join(addTags, ",")},
			}
			err := qbclient.apiPost("api/v2/torrents/removeTags", data)
			if err != nil {
				return err
			}
		}
		if len(addTags) > 0 {
			data := url.Values{
				"hashes": {infoHash},
				"tags":   {strings.Join(addTags, ",")},
			}
			err := qbclient.apiPost("api/v2/torrents/addTags", data)
			if err != nil {
				return err
			}
		}
	}

	if option.DownloadSpeedLimit != 0 && option.DownloadSpeedLimit != qbtorrent.Dl_limit {
		data := url.Values{
			"hashes": {infoHash},
			"limit":  {fmt.Sprint(option.DownloadSpeedLimit)},
		}
		err := qbclient.apiPost("api/v2/torrents/setDownloadLimit", data)
		if err != nil {
			return err
		}
	}

	if option.UploadSpeedLimit != 0 && option.UploadSpeedLimit != qbtorrent.Up_limit {
		data := url.Values{
			"hashes": {infoHash},
			"limit":  {fmt.Sprint(option.UploadSpeedLimit)},
		}
		err := qbclient.apiPost("api/v2/torrents/setUploadLimit", data)
		if err != nil {
			return err
		}
	}

	if option.Pause {
		if qbtorrent.CanPause() {
			qbclient.PauseTorrents([]string{qbtorrent.Hash})
		}
	} else if option.Resume {
		if qbtorrent.CanResume() {
			qbclient.ResumeTorrents([]string{qbtorrent.Hash})
		}
	}

	return nil
}

func (qbclient *Client) PurgeCache() {
	qbclient.data = nil
	qbclient.datatime = 0
}

func (qbclient *Client) sync() error {
	if qbclient.datatime > 0 {
		return nil
	}
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	err = qbclient.apiRequest("api/v2/sync/maindata", &qbclient.data)
	if err != nil {
		return err
	}
	qbclient.datatime = utils.Now()
	// make hash available in torrent itself as well as map key
	for hash, torrent := range qbclient.data.Torrents {
		torrent.Hash = hash
		qbclient.data.Torrents[hash] = torrent
	}
	return nil
}

func (qbclient *Client) TorrentRootPathExists(rootFolder string) bool {
	if rootFolder == "" {
		return false
	}
	err := qbclient.sync()
	if err != nil {
		return false
	}
	for _, torrent := range qbclient.data.Torrents {
		if strings.HasSuffix(torrent.Content_path, "/"+rootFolder) {
			return true
		}
	}
	return false
}

func (qbclient *Client) GetStatus() (*client.Status, error) {
	err := qbclient.sync()
	if err != nil {
		return nil, err
	}
	var status client.Status
	status.DownloadSpeed = qbclient.data.Server_state.Dl_info_speed
	status.UploadSpeed = qbclient.data.Server_state.Up_info_speed
	status.DownloadSpeedLimit = qbclient.data.Server_state.Dl_rate_limit
	status.UploadSpeedLimit = qbclient.data.Server_state.Up_rate_limit
	status.FreeSpaceOnDisk = qbclient.data.Server_state.Free_space_on_disk
	// @workaround
	// qb 的 Web API 有 bug，有时 FreeSpaceOnDisk 返回 0，但实际硬盘剩余空间充足，原因尚不明确。目前在 Windows QB 4.5.2 上发现此现象。
	if status.FreeSpaceOnDisk == 0 {
		hasDownloadingTorrent := false
		hasErrorTorrent := false
		for _, qbtorrent := range qbclient.data.Torrents {
			if qbtorrent.State == "downloading" {
				hasDownloadingTorrent = true
				break
			}
			if qbtorrent.State == "error" && qbtorrent.Completed < qbtorrent.Size {
				hasErrorTorrent = true
			}
		}
		if hasDownloadingTorrent || !hasErrorTorrent {
			status.FreeSpaceOnDisk = -1
		}
	}
	if slices.Index(qbclient.data.Tags, "_noadd") != -1 {
		status.NoAdd = true
	}
	return &status, nil
}

func (qbclient *Client) GetConfig(variable string) (string, error) {
	err := qbclient.login()
	if err != nil {
		return "", fmt.Errorf("login error: %v", err)
	}
	switch variable {
	case "global_download_speed_limit":
		v := 0
		err = qbclient.apiRequest("api/v2/transfer/downloadLimit", &v)
		return fmt.Sprint(v), err
	case "global_upload_speed_limit":
		v := 0
		err = qbclient.apiRequest("api/v2/transfer/uploadLimit", &v)
		return fmt.Sprint(v), err
	default:
		return "", nil
	}
}

func (qbclient *Client) SetConfig(variable string, value string) error {
	err := qbclient.login()
	if err != nil {
		return fmt.Errorf("login error: %v", err)
	}
	switch variable {
	case "global_download_speed_limit":
		err = qbclient.apiRequest("api/v2/transfer/setDownloadLimit?limit="+value, nil)
		return err
	case "global_upload_speed_limit":
		err = qbclient.apiRequest("api/v2/transfer/setUploadLimit?limit="+value, nil)
		return err
	default:
		return nil
	}
}

func (qbclient *Client) GetTorrent(infoHash string) (*client.Torrent, error) {
	err := qbclient.sync()
	if err != nil {
		return nil, err
	}
	qbtorrent := qbclient.data.Torrents[infoHash]
	if qbtorrent == nil {
		return nil, nil
	}
	return qbtorrent.ToTorrent(), nil
}

func (qbclient *Client) GetTorrents(stateFilter string, category string, showAll bool) ([]client.Torrent, error) {
	torrents := make([]client.Torrent, 0)
	err := qbclient.sync()
	if err != nil {
		return nil, err
	}

	for _, qbtorrent := range qbclient.data.Torrents {
		if category != "" && category != qbtorrent.Category {
			continue
		}
		if !showAll && qbtorrent.Dlspeed < 1024 && qbtorrent.Upspeed < 1024 {
			continue
		}
		torrent := qbtorrent.ToTorrent()
		if !torrent.MatchStateFilter(stateFilter) {
			continue
		}
		torrents = append(torrents, *torrent)
	}
	return torrents, nil
}

func (qbclient *Client) GetTorrentContents(infoHash string) ([]client.TorrentContentFile, error) {
	err := qbclient.login()
	if err != nil {
		return nil, fmt.Errorf("login error: %v", err)
	}
	apiUrl := qbclient.ClientConfig.Url + "api/v2/torrents/files?hash=" + infoHash
	qbTorrentContents := []apiTorrentContent{}
	err = utils.FetchJson(apiUrl, &qbTorrentContents, qbclient.HttpClient, "", "", nil)
	if err != nil {
		return nil, err
	}
	torrentContents := []client.TorrentContentFile{}
	for _, qbTorrentContent := range qbTorrentContents {
		torrentContents = append(torrentContents, client.TorrentContentFile{
			Index:    qbTorrentContent.Index,
			Path:     strings.ReplaceAll(qbTorrentContent.Name, "\\", "/"),
			Size:     qbTorrentContent.Size,
			Complete: qbTorrentContent.Is_seed,
		})
	}
	return torrentContents, nil
}

func (qbclient *Client) GetTorrentTrackers(infoHash string) ([]client.TorrentTracker, error) {
	err := qbclient.login()
	if err != nil {
		return nil, fmt.Errorf("login error: %v", err)
	}
	apiUrl := qbclient.ClientConfig.Url + "api/v2/torrents/trackers?hash=" + infoHash
	qbTorrentTrackers := []apiTorrentTracker{}
	err = utils.FetchJson(apiUrl, &qbTorrentTrackers, qbclient.HttpClient, "", "", nil)
	if err != nil {
		return nil, err
	}
	trackers := utils.Map(qbTorrentTrackers, func(qbtracker apiTorrentTracker) client.TorrentTracker {
		status := ""
		switch qbtracker.Status {
		case 0:
			status = "disabled"
		case 1:
			status = "notcontacted"
		case 2:
			status = "working"
		case 3:
			status = "updating"
		case 4:
			status = "error"
		default:
			status = "unknown"
		}
		return client.TorrentTracker{
			Url:    qbtracker.Url,
			Msg:    qbtracker.Msg,
			Status: status,
		}
	})
	return trackers, nil
}

func (qbclient *Client) EditTorrentTracker(infoHash string, oldTracker string, newTracker string, replaceHost bool) error {
	if replaceHost {
		torrent, err := qbclient.GetTorrent(infoHash)
		if err != nil {
			return err
		}
		if torrent == nil {
			return fmt.Errorf("torrent %s not found", infoHash)
		}
		trackers, err := qbclient.GetTorrentTrackers(torrent.InfoHash)
		if err != nil {
			return fmt.Errorf("failed to get torrent %s trackers: %v", torrent.InfoHash, err)
		}
		oldTrackerUrl := ""
		newTrackerUrl := ""
		directNewUrlMode := utils.IsUrl(newTracker)
		for _, tracker := range trackers {
			oldTrackerUrlObj, err := url.Parse(tracker.Url)
			if err != nil {
				continue
			}
			if oldTrackerUrlObj.Host == oldTracker {
				oldTrackerUrl = tracker.Url
				if directNewUrlMode {
					newTrackerUrl = newTracker
					break
				}
				oldTrackerUrlObj.Host = newTracker
				newTrackerUrl = oldTrackerUrlObj.String()
				break
			}
		}
		if oldTrackerUrl != "" && newTrackerUrl != "" {
			if oldTrackerUrl == newTrackerUrl {
				return nil
			}
			err := qbclient.EditTorrentTracker(torrent.InfoHash, oldTrackerUrl, newTrackerUrl, false)
			if err != nil {
				log.Errorf("Failed to replace torrent %s tracker domain: %v", torrent.InfoHash, err)
			} else {
				log.Debugf("Replaced torrent %s tracker %s => %s", torrent.InfoHash, oldTrackerUrl, newTrackerUrl)
			}
			return err
		}
		return fmt.Errorf("torrent %s old tracker does NOT exist", torrent.InfoHash)
	}
	data := url.Values{
		"hash":    {infoHash},
		"origUrl": {oldTracker},
		"newUrl":  {newTracker},
	}
	return qbclient.apiPost("api/v2/torrents/editTracker", data)
}

func (qbclient *Client) AddTorrentTrackers(infoHash string, trackers []string) error {
	data := url.Values{
		"hash": {infoHash},
		"urls": {strings.Join(trackers, "\n")},
	}
	return qbclient.apiPost("api/v2/torrents/addTrackers", data)
}

func (qbclient *Client) RemoveTorrentTrackers(infoHash string, trackers []string) error {
	data := url.Values{
		"hash": {infoHash},
		"urls": {strings.Join(trackers, "|")},
	}
	return qbclient.apiPost("api/v2/torrents/removeTrackers", data)
}

func (qbclient *Client) Close() {
	if qbclient.Logined {
		qbclient.apiPost("api/v2/auth/logout", nil)
		qbclient.Logined = false
	}
	qbclient.PurgeCache()
}

func NewClient(name string, clientConfig *config.ClientConfigStruct, config *config.ConfigStruct) (client.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &Client{
		Name:         name,
		ClientConfig: clientConfig,
		Config:       config,
		NoLogin:      clientConfig.QbittorrentNoLogin,
		HttpClient: &http.Client{
			Jar: jar,
		},
	}
	return client, nil
}

func init() {
	client.Register(&client.RegInfo{
		Name:    "qbittorrent",
		Creator: NewClient,
	})
}

var (
	_ client.Client = (*Client)(nil)
)
