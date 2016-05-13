package core

import (
	"encoding/json"
	"net/http"
	//"net/url"
	"os"
	"os/signal"
	//"strconv"
	"strings"
	"sync"
	"syscall"
	//"strings"
	"bytes"
	"core/config"
	"core/util"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"time"
)

type AdViewServer struct {
	Conf *config.ItemConf
	//查询接口
	QueryMux *HandlerMux

	//数据channel
	data_chan chan string

	//quit channel
	quit_chan chan int
}

var _ = fmt.Println

var g_waiter sync.WaitGroup

func StartServer() {
	server := NewServer()
	server.Start()
}

func GetIpFromHostPort(addr string) string {
	pos := strings.Index(addr, ":")
	if pos > 0 {
		return addr[:pos]
	}
	return addr
}

//构造函数
func NewServer() *AdViewServer {
	g_conf := config.NewItemConf("core/config/adview.conf")

	//初始化quit_chan
	quit_chan := make(chan int)

	//Mux Initialization
	query_mux := make(HandlerMux)

	server := &AdViewServer{
		Conf:      g_conf,
		QueryMux:  &query_mux,
		data_chan: make(chan string, 100000),
		quit_chan: quit_chan,
	}
	util.Info("AdViewServer Initialize.....")
	fmt.Println("AdViewServer Initialize.....")
	server.Init()

	return server
}

func StripPrefixHandler(prefix string, h http.Handler) http.HandlerFunc {
	if prefix == "" {
		return h.ServeHTTP
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			fmt.Println("path=", p)
			r.URL.Path = p
			h.ServeHTTP(w, r)
		} else {
			p := strings.TrimPrefix(r.URL.Path, prefix)
			fmt.Println("path=", p)
			http.NotFound(w, r)
		}
	})
}

func (self *AdViewServer) Init() {
	//初始化Query接口
	(*self.QueryMux)["/adview/adlink"] = self.AdViewLinkService
	(*self.QueryMux)["/360/adlink"] = self.Ad360LinkService
	js_fs := http.FileServer(http.Dir("core/js"))
	(*self.QueryMux)["/js"] = http.StripPrefix("/js/", js_fs).ServeHTTP
	html_fs := http.FileServer(http.Dir("core/html"))
	(*self.QueryMux)["/html"] = http.StripPrefix("/html/", html_fs).ServeHTTP

	go self.StartJob()
}

//后台线程
func (self *AdViewServer) StartJob() {
	g_waiter.Add(1)
	defer g_waiter.Done()

	queues := make([]string, 0)
	syncTicker := time.NewTicker(2 * time.Minute)
Done:
	for {
		select {
		case req := <-self.data_chan:
			queues = append(queues, req)
			if len(queues) > 50 {
				//Save it
				self.Save(queues)
				queues = make([]string, 0)
			}
		case <-syncTicker.C:
			if len(queues) > 0 {
				self.Save(queues)
				queues = make([]string, 0)
			}
		case <-self.quit_chan:
			break Done
		}
	}
	self.Save(queues)
}

func (self *AdViewServer) Save(queues []string) {
	if len(queues) <= 0 {
		return
	}

	res := strings.Join(queues, "\n")
	ts := time.Now().Format("2006010215")
	fn := "core/log/log_" + ts + ".log"
	fd, err := os.OpenFile(fn, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		util.Error("写入文件错误:file=%s,%s\n", fn, err.Error())
		return
	}
	defer fd.Close()
	//fd.Write([]byte(res + "\n"))
	fd.Write([]byte(res))
}

//adrequest
func (self *AdViewServer) HttpDo(name string, url string, adreq_json []byte, w http.ResponseWriter, r *http.Request) {
	/*adreq_json = []byte(`{
	  "bid": "bdf3fd3c522e310f",
	  "app": {
	    "app_name": "测试DEMO12",
	    "package_name": "com.mediav.ads",
	    "category": 1000,
	    "app_version": "1.0"
	  },
	  "device" : {
	    "device_id" : [{
	      "device_id": "a0000fddfsd",
	      "device_id_type":1,
	      "hash_type":0   //NONE
	    },
	    {
	      "device_id": "a0000fddfs2",
	      "device_id_type":2,
	      "hash_type":0   //NONE
	    }
	    ],
	    "os_type": 2, // OS_ANDROID
	    "os_version": "6.0.1",
	    "brand":"Xiaomi",
	    "model":"2014813",
	    "device_type":1,  // PHONE
	    "screen_width":1280,
	    "screen_height":960,
	    "screen_density":2.5, // 屏幕密度
	    "screen_orientation":2, // 横向
	    "carrier_id":70120 // CHINA_MOBILE 移动
	  },
	  "adspaces" : [{
	    "adspace_id":"PPub5d0djn",
	    "adspace_type":4, // 信息流
	    "adspace_position":1, // 首屏
	    "allowed_html":false,  // 不支持html创意
	    "width":320,
	    "height":50,
	    "impression_num":1, // 该广告位返回一个创意
	    "keywords":"姚明",
	    "channel":"体育",
	    "open_type":2, // 外开
	    "interaction_type": [
	        2, // 浏览
	        3 // 下载
	        ]
	  }],
	  "uid":"85e9167c065e4b14fab5711b6a1b0786",
	  "ip":"210.221.10.2",
	  "network_type":1, //wifi
	  "user_agent":"Dalvik/2.1.0 (Linux; U; Android 6.0.1; 2014813 Build/MHC19Q)",
	  "longitude":115.234521,
	  "latitude":90.322343
	}`)
	fmt.Println("--------------------------------")
	fmt.Println(string(adreq_json))*/
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(adreq_json))
	req.Header.Set("Content-Type", "application/json")
	if strings.Contains(name, "adview/adlink") {
		req.Header.Set("x-adviewssp-version", "2.3")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)

	if strings.Contains(name, "adview/adlink") {
		adresp := AdResponse{}
		json.Unmarshal(body, &adresp)
		for _, ad := range adresp.Ads {
			//fmt.Println("response Body:", ad.GetAdm())
			html := "<html>"
			html += ad.GetAdm()
			html += "</html>"
			w.Write([]byte(html))
		}
	} else if strings.Contains(name, "360/adlink") {
		juxiaoresp := BidResponse{}
		json.Unmarshal(body, &juxiaoresp)
		//fmt.Println(string(body))
		for _, ad := range juxiaoresp.Ads {
			for _, creative := range ad.GetCreative() {

				/*html := "<html>"
				html += "<title>" + creative.GetAdm().GetNative().GetTitle().GetText() + "</title>"
				html += "<script type='text/javascript'>function onshow(){alert('load');}</script>"
				html += "<body onload='onshow();'>"
				html += "<a href='" + creative.GetInteractionObject().GetUrl()
				html += "'><img src='" + creative.GetAdm().GetNative().GetImg().GetUrl()
				html += "'></a></body></html>"*/
				//fmt.Println(creative.GetInteractionObject().GetUrl())
				//fmt.Println(creative.GetAdm().GetNative().GetImg().GetUrl())
				event_map := make(map[string][]string)
				for _, event := range creative.GetEventTrack() {
					event_map[event.GetEventType().String()] = event.GetNotifyUrl()
				}

				data := struct {
					Title           string
					Interaction_Url string
					Image           string
					Event           map[string][]string
				}{
					Title:           creative.GetAdm().GetNative().GetTitle().GetText(),
					Interaction_Url: creative.GetInteractionObject().GetUrl(),
					Image:           creative.GetAdm().GetNative().GetImg().GetUrl(),
					Event:           event_map,
				}

				t, _ := template.New("adtmpl.html").ParseFiles("core/html/adtmpl.html")

				err := t.Execute(w, data)
				if err != nil {
					fmt.Println(err)
				}
				//w.Write([]byte(html))
			}
		}

	} else {
		w.Write([]byte("404 Page not found!"))
	}
	self.Save([]string{string(body)})

}

//处理入口
func (self *AdViewServer) Handle(name string, adreq_jsop []byte, ad_link string, w http.ResponseWriter, r *http.Request) {

	self.HttpDo(name, ad_link, adreq_jsop, w, r)

}

func (self *AdViewServer) ParamParse(r *http.Request) *AdRequest {
	base64_str := r.FormValue("Mobile_info")
	req_json, _ := base64.StdEncoding.DecodeString(base64_str)

	//fmt.Println(string(req_json))

	app_id := "SDK20161506030508iumt55xlwjf86v3"
	app_name := "万能汇率"
	app_domain := "itunes.apple.com"
	app_cat := []int32{10404}
	app_ver := "1.2.0.1"
	app_bundle := "com.appdream.DisWanNengHuiLv"
	app_paid := int32(0)
	app_kw := "财务"
	app_storeurl := "https://itunes.apple.com/cn/app/wan-neng-hui-lu/id433845696?mt=8"

	/*app_id := "SDK20161506030517poxrml3ivy20351"
	app_name := "迷你卸载"
	app_domain := "apk.hiapk.com"
	app_cat := []int32{10404}
	app_ver := "1.4.3"
	app_bundle := "com.mini.android.uninstaller"
	app_paid := int32(0)
	app_kw := "系统软件系统类别"
	app_storeurl := "http://apk.hiapk.com/appdown/com.mini.android.uninstaller"*/

	app := AdRequest_App{
		Id:       &app_id,
		Name:     &app_name,
		Domain:   &app_domain,
		Cat:      app_cat,
		Ver:      &app_ver,
		Bundle:   &app_bundle,
		Paid:     &app_paid,
		Keywords: &app_kw,
		Storeurl: &app_storeurl,
	}

	adreq := AdRequest{
		App: &app,
	}
	json.Unmarshal(req_json, &adreq)
	return &adreq

}

//处理adview
func (self *AdViewServer) AdViewLinkService(w http.ResponseWriter, r *http.Request) {
	adreq := self.ParamParse(r)
	//fmt.Println("--------------------------------------------")
	//fmt.Println(adreq.App.String())
	adreq_jsop, _ := json.Marshal(adreq)
	self.Handle("adview/adlink", adreq_jsop, "http://bid.adview.cn/agent/sspReqAd", w, r)
}

//生成32位md5字串
func (self *AdViewServer) GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

//处理360聚效
func (self *AdViewServer) Ad360LinkService(w http.ResponseWriter, r *http.Request) {
	adreq := self.ParamParse(r)
	/*app := BidRequest_App{
		AppName:     adreq.App.Name,
		PackageName: adreq.App.Bundle,
		Category:    &adreq.App.Cat[0],
		AppVersion:  adreq.App.Ver,
	}*/
	appname := "测试DEMO"
	apppackage := "com.mediav.ads"
	category := int32(1000)
	appver := "1.0"
	app := BidRequest_App{
		AppName:     &appname,
		PackageName: &apppackage,
		Category:    &category,
		AppVersion:  &appver,
	}
	/*imei_idfa := BidRequest_DeviceId_DEVICE_UNKNOWN
	hashtype := BidRequest_DeviceId_NONE
	deviceId := ""
	if adreq.Device.GetOs() == "iOS" {
		imei_idfa = BidRequest_DeviceId_IDFV
		deviceId = adreq.GetDevice().GetDpid()
	} else {
		imei_idfa = BidRequest_DeviceId_IMEI
		hashtype = BidRequest_DeviceId_MD5
		deviceId = adreq.GetDevice().GetDid()
	}

	deviceid := BidRequest_DeviceId{
		DeviceId:     &deviceId,
		DeviceIdType: &imei_idfa,
		HashType:     &hashtype,
	}
	var ostype BidRequest_Device_OS_Type = 0
	if adreq.Device.GetOs() == "iOS" {
		ostype = BidRequest_Device_OS_IOS
	} else {
		ostype = BidRequest_Device_OS_ANDROID
	}

	devicetype := BidRequest_Device_Device_Type(adreq.Device.GetDevicetype())
	if devicetype == 1 || devicetype == 2 || devicetype == 4 {
		devicetype = BidRequest_Device_PHONE
	} else if devicetype == 0 {
		devicetype = BidRequest_Device_UNKNOWN
	} else {
		devicetype = BidRequest_Device_TABLET
	}

	density := float64(adreq.Device.GetSDensity())
	orientation := BidRequest_Device_Screen_Orientation(adreq.Device.GetOrientation())
	carrier := BidRequest_Device_CHINA_MOBILE
	*/

	deviceId1 := "a0000fddfsd"
	deviceId2 := "a0000fddfs2"
	device_id_type := BidRequest_DeviceId_Device_Id_Type(1)
	hash_type := BidRequest_DeviceId_Hash_Type(0)

	deviceid_1 := BidRequest_DeviceId{
		DeviceId:     &deviceId1,
		DeviceIdType: &device_id_type,
		HashType:     &hash_type,
	}

	deviceid_2 := BidRequest_DeviceId{
		DeviceId:     &deviceId2,
		DeviceIdType: &device_id_type,
		HashType:     &hash_type,
	}

	var ostype BidRequest_Device_OS_Type = 2
	osv := "6.0.1"                                  //adreq.Device.Osv
	brand := "Xiaomi"                               //adreq.Device.Make
	model := "2014813"                              //adreq.Device.Model,
	device_type := BidRequest_Device_Device_Type(1) // PHONE
	screen_width := int32(1280)
	screen_height := int32(960)
	screen_density := float64(2.5)                                // 屏幕密度
	screen_orientation := BidRequest_Device_Screen_Orientation(2) // 横向
	carrier_id := BidRequest_Device_Carrier_Id(70120)             // CHINA_MOBILE 移动

	device := BidRequest_Device{
		DeviceId:          []*BidRequest_DeviceId{&deviceid_1, &deviceid_2},
		OsType:            &ostype,
		OsVersion:         &osv,   //adreq.Device.Osv,
		Brand:             &brand, //adreq.Device.Make,
		Model:             &model, //adreq.Device.Model,
		DeviceType:        &device_type,
		ScreenWidth:       &screen_width,  //adreq.Device.Sw,
		ScreenHeight:      &screen_height, //adreq.Device.Sh,
		ScreenDensity:     &screen_density,
		ScreenOrientation: &screen_orientation,
		CarrierId:         &carrier_id, //adreq.Device.GetCarrier(),
	}
	/*impNum := int32(1)
	kw := ""
	channel := ""
	adspace_type := BidRequest_Adspaces_Adspace_Type(adreq.Imp[0].GetInstl())
	pos := BidRequest_Adspaces_Adspace_Position(adreq.Imp[0].GetBanner().GetPos())
	boo := true
	opentype := BidRequest_Adspaces_ALL*/

	adspace_type := BidRequest_Adspaces_Adspace_Type(4)         // 信息流
	adspace_position := BidRequest_Adspaces_Adspace_Position(1) // 首屏
	allowed_html := false                                       // 不支持html创意
	width := int32(320)
	height := int32(50)
	impression_num := int32(1) // 该广告位返回一个创意
	keywords := "姚明"
	channel := "体育"
	open_type := BidRequest_Adspaces_Open_Type(2) // 外开
	interaction_type := []BidRequest_Adspaces_Interaction_Type{2, 3}

	adspace_id := "PPub5d0djn"
	adspaces := BidRequest_Adspaces{
		AdspaceId:       &adspace_id,
		AdspaceType:     &adspace_type,
		AdspacePosition: &adspace_position,
		AllowedHtml:     &allowed_html,
		Width:           &width,  //adreq.Imp[0].GetBanner().W,
		Height:          &height, //adreq.Imp[0].GetBanner().H,
		ImpressionNum:   &impression_num,
		Keywords:        &keywords,
		Channel:         &channel,
		OpenType:        &open_type,
		InteractionType: interaction_type, //[]BidRequest_Adspaces_Interaction_Type{BidRequest_Adspaces_ANY},
	}
	//uid生成规则如下:
	//uid = MD5(设备号 + package_name + 页面属性)
	//页面属性一般选择能唯一标识app页面的属性, 比如页面对应的activity。
	str := adreq.GetDevice().GetDid() + adreq.GetApp().GetBundle()
	uid := self.GetMd5String(str)
	nettype := BidRequest_NET_UNKNOWN
	lon := float64(adreq.GetDevice().GetGeo().GetLon())
	lat := float64(adreq.GetDevice().GetGeo().GetLat())
	userAgent := adreq.GetDevice().GetUa()
	bidreq := BidRequest{
		Bid:         adreq.Id,
		App:         &app,
		Device:      &device,
		Adspaces:    []*BidRequest_Adspaces{&adspaces},
		Uid:         &uid,
		Ip:          adreq.GetDevice().Ip,
		UserAgent:   &userAgent,
		NetworkType: &nettype,
		Longitude:   &lon,
		Latitude:    &lat,
	}
	bidreq_jsop, _ := json.Marshal(bidreq)
	//fmt.Println(string(bidreq_jsop))

	self.Handle("360/adlink", bidreq_jsop, "http://test.m.mdvdns.com/a?type=2", w, r)

}

//全局channel
var pa_quit_chan = make(chan int)

//启动主程序
func (server *AdViewServer) Start() bool {
	util.Info("AdViewServer starting.......")
	//创建交互信号
	//退出信号
	//quit_chan := make(chan int)
	quit_chan := server.quit_chan
	query_chan := make(chan int)

	//gracefully exit when ctrl-c is pressed
	k := make(chan os.Signal, 1)
	signal.Notify(k, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		//等待信号
		<-k
		//告诉所有进程退出
		close(quit_chan)
		close(pa_quit_chan)
		util.Info("AdViewServer going to Exiting")
	}()

	//启动查询服务
	//Start A Stoppable Http Service
	StartHttpServer(server.QueryMux, server.Conf.QueryHost, server.Conf.QueryPort, quit_chan, query_chan)

	//等待程序结束
	<-query_chan

	//Wait all things to exit gracefully
	g_waiter.Wait()

	util.Info("AdViewServer stopped....\n")
	util.Flush()
	return true
}
