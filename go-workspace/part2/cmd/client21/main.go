package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	pb "os-project/part2/imgdownload"
	"os-project/part2/shmatomicint"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type statistics struct {
	me_download      bool
	img_url          string
	server_addr      string
	handle_latency   int64
	inserver_latency int64
	rtt_latency      int64
	total_latency    int64
	ongoing_tasks    int32
}

var (
	// Command line arguments
	threads    = flag.Int("n-t", 1, "Number of threads to use")
	is_master  = flag.Bool("master", false, "Master: Start the shm integer")
	out_dir    = flag.String("out-dir", "/osdata/osgroup4/download_imgs/", "output directory")
	stats_file = flag.String("stats-file", "stats.csv", "output statistics file")
	time_file  = flag.String("time-file", "time.txt", "output time file")
	proc_id    = flag.Int("proc-id", 1, "process id")

	// Shared memory
	shm_name = "shm_atomic_int"
	err      error
	task_no  *shmatomicint.ShmAtomicInt

	// grpc
	serverAddr = flag.String("addr", "localhost:51151", "The server address in the format of host:port")
	opts       []grpc.DialOption
	conn       *grpc.ClientConn

	// others
	wg    sync.WaitGroup
	stats [2000]statistics
)

func main() {
	flag.Parse()

	if *is_master {
		*proc_id = 0
		log.Printf("Master: Creating shm...")
		task_no, err = shmatomicint.New(shm_name, 0)
		if err != nil {
			log.Fatalf("Error creating shm atomic int: %v", err)
		}
		defer task_no.Unlink()
	} else {
		err = errors.New("Nothing")
		if *proc_id == 0 || *proc_id > 47 {
			log.Printf("Error: non master proc-id cannot be 0 or greater than 47")
			return
		}
		// must wait for master to create shm
		// otherwise, we will be stuck in a loop
		log.Printf("Proc %d: Waiting for master to create shm...", *proc_id)
		log.Printf("Remember that the ids of non master processes shall be continuous and starting from 1")
		for err != nil {
			task_no, err = shmatomicint.Bind(shm_name)
		}
	}

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	log.Printf("Dialing %s...", *serverAddr)
	conn, err = grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Printf("fail to dial: %v", err)
		return // Use return to perform task_no.Unlink() in defer
	}
	defer conn.Close()

	client := pb.NewImgDownloadClient(conn)

	wg.Add(*threads)
	log.Printf("Proc %d: Start downloading images...", *proc_id)
	begin := time.Now()
	for i := 0; i < *threads; i++ {
		go func() {
			defer wg.Done()
			for task := task_no.AtomicFetchAdd(1); task < 2000; task = task_no.AtomicFetchAdd(1) {
				// request the image
				// log.Printf("Task %d: File %s.JPEG\n", task, requests[task])
				stats[task].me_download = true
				start := time.Now()
				res, err := client.GetSingleImg(context.Background(), &pb.ImgRequest{
					Url: requests[task],
					Sz:  &pb.Size{Width: 128, Height: 128},
				})

				if err != nil {
					log.Printf("Error getting image %s: %v", requests[task], err)
					continue
				}

				rtt_latency := time.Since(start).Nanoseconds()

				// Save the image now
				// log.Printf("Task %d: File %s.JPEG downloaded, Ongoing task %d\n", task, requests[task], res.OngoingRequests)
				b := res.Img
				filename := filepath.Join(*out_dir, requests[task]+".JPEG")
				file, _ := os.Create(filename)
				file.Write(b)
				total_latency := time.Since(start).Nanoseconds()

				stats[task] = statistics{
					me_download:      true,
					img_url:          requests[task],
					server_addr:      *serverAddr,
					handle_latency:   res.HandleLatency,
					inserver_latency: res.InserverLatency,
					rtt_latency:      rtt_latency,
					total_latency:    total_latency,
					ongoing_tasks:    res.OngoingRequests,
				}
			}
		}()
	}

	wg.Wait()
	log.Printf("Proc %d: All images downloaded.", *proc_id)
	total_time := time.Since(begin).Nanoseconds()
	log.Printf("Proc %d: Total time: %d", *proc_id, total_time)

	// Save the statistics
	var file, time_f *os.File
	if *is_master {
		file, err = os.Create(*stats_file)
		if err != nil {
			task_no.AtomicStore(99999)
			return
		}

		time_f, err = os.Create(*time_file)
		if err != nil {
			task_no.AtomicStore(99999)
			return
		}

		task_no.AtomicStore(3000)
		file.WriteString("img_url,server_addr,handle_latency,inserver_latency,rtt_latency,total_latency,ongoing_tasks\n")
	} else {
		sig := task_no.AtomicLoad()
		for {
			if sig == 99999 {
				return
			} else if sig >= 3000 {
				break
			}
			sig = task_no.AtomicLoad()
		}
	}

	var id int
	for id = task_no.AtomicLoad() - 3000; id < *proc_id*100; id = task_no.AtomicLoad() - 3000 {
		time.Sleep(10 * time.Millisecond)
	}

	if id >= (*proc_id+1)*100 {
		log.Printf("Proc %d: Master cannot create the file.", *proc_id)
		return
	}

	if !*is_master {
		file, _ = os.OpenFile(*stats_file, os.O_APPEND|os.O_WRONLY, 0644)
		time_f, _ = os.OpenFile(*time_file, os.O_APPEND|os.O_WRONLY, 0644)
	}

	log.Printf("Proc %d: Saving statistics...", *proc_id)

	for i := 0; i < 2000; i++ {
		if !stats[i].me_download {
			continue
		}
		file.WriteString(stats[i].img_url + "," + stats[i].server_addr + "," +
			strconv.FormatInt(stats[i].handle_latency, 10) + "," +
			strconv.FormatInt(stats[i].inserver_latency, 10) + "," +
			strconv.FormatInt(stats[i].rtt_latency, 10) + "," +
			strconv.FormatInt(stats[i].total_latency, 10) + "," +
			strconv.FormatInt(int64(stats[i].ongoing_tasks), 10) + "\n")
	}
	time_f.WriteString(strconv.FormatInt(total_time, 10) + "\n")

	// Flush the file
	file.Close()
	time_f.Close()

	task_no.AtomicFetchAdd(100) // Why we add 100 at a time?
	// Because the previous working goroutines may accidentally add
	// one to task_no, which may lead to an unexpceted behaviour.

	log.Printf("Proc %d: Statistics saved.", *proc_id)
}

// The all reqeusts are hard-coded here, 2000 requests in total, 19 requests per line
var requests = []string{
	"vlwf34", "vlwf00", "vlwx46", "vlwx07", "vl3e03", "vl3e08", "vl2i42", "vl2i21", "vlmt22", "vlmt14", "vl8618", "vl8636", "vljq19", "vljq34", "vlym45", "vlym13", "vlgi03", "vlgi41", "vljd47",
	"vljd11", "vlpb46", "vlpb32", "vl2c31", "vl2c17", "vlpw07", "vlpw03", "vlk223", "vlk248", "vlls16", "vlls36", "vlgh41", "vlgh11", "vl5422", "vl5444", "vl0812", "vl0836", "vlhv32", "vlhv28",
	"vl6028", "vl6032", "vlpa17", "vlpa42", "vlne43", "vlne09", "vl7s15", "vl7s20", "vl3t13", "vl3t22", "vlmc48", "vlmc19", "vl0724", "vl0717", "vljl37", "vljl12", "vliq43", "vliq49", "vlvi47",
	"vlvi13", "vlv647", "vlv607", "vlfq31", "vlfq38", "vl0z30", "vl0z42", "vl5x42", "vl5x03", "vl1n40", "vl1n36", "vl8w34", "vl8w47", "vlr634", "vlr606", "vl0d06", "vl0d29", "vlv908", "vlv906",
	"vlaj47", "vlaj02", "vl0904", "vl0944", "vlqh44", "vlqh14", "vlu946", "vlu902", "vl2u33", "vl2u28", "vljz18", "vljz25", "vllf40", "vllf34", "vllb34", "vllb35", "vl1e36", "vl1e47", "vlxl24",
	"vlxl47", "vlax15", "vlax47", "vl1016", "vl1034", "vlxa03", "vlxa43", "vl0c18", "vl0c04", "vl1a27", "vl1a39", "vl8000", "vl8048", "vliv28", "vliv30", "vl9234", "vl9212", "vlts16", "vlts22",
	"vlrf25", "vlrf29", "vlql14", "vlql38", "vlgz44", "vlgz34", "vl2f22", "vl2f42", "vlfg46", "vlfg35", "vle727", "vle714", "vlad45", "vlad19", "vll923", "vll922", "vl0547", "vl0527", "vlmq27",
	"vlmq48", "vlhh08", "vlhh49", "vlup40", "vlup27", "vl4w39", "vl4w42", "vlya42", "vlya02", "vlyx23", "vlyx35", "vlci38", "vlci45", "vlsv25", "vlsv23", "vljo45", "vljo38", "vlez45", "vlez36",
	"vlw227", "vlw221", "vlvv18", "vlvv20", "vlbi20", "vlbi30", "vln918", "vln949", "vl4143", "vl4104", "vloh41", "vloh34", "vli916", "vli908", "vlj706", "vlj722", "vlct40", "vlct26", "vlix08",
	"vlix28", "vltv27", "vltv49", "vl5z33", "vl5z29", "vlx133", "vlx143", "vl3b12", "vl3b43", "vlox21", "vlox38", "vlfi05", "vlfi39", "vl9c05", "vl9c23", "vl9t18", "vl9t16", "vld830", "vld819",
	"vl7102", "vl7101", "vlmp34", "vlmp14", "vlkq03", "vlkq34", "vla141", "vla100", "vl4f03", "vl4f40", "vln145", "vln111", "vl5f42", "vl5f01", "vltk08", "vltk28", "vl4m19", "vl4m08", "vlr117",
	"vlr102", "vlrv10", "vlrv31", "vlrg48", "vlrg17", "vlma24", "vlma31", "vlyf11", "vlyf37", "vlbg42", "vlbg20", "vles09", "vles12", "vl5038", "vl5019", "vl0237", "vl0239", "vlf636", "vlf645",
	"vluo33", "vluo14", "vl8117", "vl8111", "vlza35", "vlza39", "vlas37", "vlas14", "vl7j40", "vl7j33", "vl9n11", "vl9n27", "vlfx46", "vlfx12", "vlbb14", "vlbb38", "vlep26", "vlep31", "vlj233",
	"vlj230", "vlkr16", "vlkr10", "vltu07", "vltu15", "vlul49", "vlul31", "vlb816", "vlb848", "vl2149", "vl2147", "vlkm20", "vlkm29", "vll815", "vll844", "vlwb22", "vlwb24", "vlml32", "vlml10",
	"vlpg46", "vlpg16", "vllq20", "vllq23", "vll644", "vll631", "vl9308", "vl9334", "vly400", "vly424", "vlcl15", "vlcl16", "vlot19", "vlot46", "vlce29", "vlce07", "vlf010", "vlf011", "vl0602",
	"vl0618", "vlwt42", "vlwt17", "vlqg17", "vlqg10", "vlrq21", "vlrq39", "vlgj04", "vlgj46", "vl5648", "vl5602", "vl1y40", "vl1y29", "vlq645", "vlq606", "vlmk47", "vlmk00", "vlsr39", "vlsr00",
	"vl9804", "vl9829", "vl3q48", "vl3q20", "vlz036", "vlz046", "vl1o20", "vl1o42", "vlti30", "vlti48", "vlc101", "vlc146", "vlxs18", "vlxs48", "vlrk23", "vlrk14", "vljh04", "vljh46", "vlki21",
	"vlki09", "vli604", "vli634", "vlwd07", "vlwd20", "vlt739", "vlt708", "vlbt43", "vlbt17", "vlpk31", "vlpk07", "vlh533", "vlh536", "vlvq03", "vlvq49", "vloe39", "vloe30", "vlmb49", "vlmb44",
	"vls915", "vls919", "vlpt43", "vlpt23", "vllc31", "vllc49", "vl6f24", "vl6f47", "vloc02", "vloc01", "vlky44", "vlky40", "vlj843", "vlj817", "vliy10", "vliy32", "vlou41", "vlou08", "vlo413",
	"vlo406", "vlu545", "vlu532", "vlb012", "vlb010", "vl9i06", "vl9i30", "vlhy49", "vlhy23", "vlcs45", "vlcs13", "vl9515", "vl9542", "vlgs47", "vlgs11", "vllu05", "vllu38", "vlbl11", "vlbl06",
	"vlon48", "vlon25", "vl1423", "vl1427", "vl9d00", "vl9d15", "vl8v32", "vl8v24", "vlqv21", "vlqv05", "vlxk01", "vlxk20", "vlkx39", "vlkx30", "vl1136", "vl1109", "vlh402", "vlh431", "vlj408",
	"vlj443", "vl0147", "vl0139", "vlbn14", "vlbn26", "vlwm36", "vlwm03", "vlzj44", "vlzj19", "vlmx40", "vlmx47", "vlle21", "vlle14", "vltp17", "vltp20", "vlue13", "vlue19", "vlnc08", "vlnc49",
	"vlr911", "vlr949", "vlfz30", "vlfz19", "vla507", "vla519", "vl1j41", "vl1j43", "vl4k29", "vl4k15", "vlna09", "vlna15", "vlyj04", "vlyj23", "vluj08", "vluj46", "vl3x13", "vl3x43", "vlp312",
	"vlp316", "vl5i24", "vl5i34", "vl9647", "vl9610", "vl2q24", "vl2q15", "vld141", "vld100", "vllt46", "vllt24", "vlr510", "vlr521", "vle105", "vle136", "vli814", "vli842", "vla714", "vla725",
	"vlx927", "vlx930", "vldq01", "vldq16", "vlhn13", "vlhn16", "vl6u29", "vl6u39", "vl9726", "vl9736", "vlag30", "vlag39", "vlsx30", "vlsx06", "vl8s05", "vl8s03", "vl6923", "vl6909", "vlrd44",
	"vlrd12", "vlpd26", "vlpd39", "vlph14", "vlph23", "vl6r03", "vl6r39", "vl7z11", "vl7z23", "vlo929", "vlo901", "vlee43", "vlee45", "vl5m30", "vl5m09", "vlpr36", "vlpr32", "vllg31", "vllg37",
	"vlwg47", "vlwg10", "vls022", "vls039", "vl4422", "vl4416", "vled20", "vled30", "vlf924", "vlf946", "vlqn28", "vlqn36", "vlr405", "vlr401", "vld019", "vld031", "vl7y21", "vl7y10", "vlnt00",
	"vlnt34", "vl4904", "vl4912", "vlt407", "vlt431", "vlkv11", "vlkv22", "vln420", "vln440", "vl4044", "vl4033", "vli310", "vli308", "vlq505", "vlq501", "vl4714", "vl4713", "vlqy05", "vlqy33",
	"vl5t28", "vl5t29", "vl9o36", "vl9o30", "vlje44", "vlje33", "vlvj32", "vlvj35", "vlrs15", "vlrs03", "vl2y49", "vl2y22", "vlaa45", "vlaa11", "vllh23", "vllh12", "vls517", "vls510", "vliu07",
	"vliu19", "vl2329", "vl2339", "vlgq36", "vlgq01", "vly138", "vly105", "vlmm31", "vlmm33", "vl9915", "vl9928", "vlac21", "vlac33", "vlum22", "vlum25", "vlrc31", "vlrc47", "vl3m05", "vl3m43",
	"vl5p06", "vl5p37", "vlkj12", "vlkj48", "vlw911", "vlw929", "vlpu41", "vlpu05", "vldk47", "vldk06", "vloi08", "vloi20", "vlcj41", "vlcj21", "vlxg33", "vlxg08", "vlg324", "vlg344", "vlp633",
	"vlp630", "vlkb46", "vlkb02", "vlia30", "vlia13", "vlx729", "vlx712", "vlo542", "vlo516", "vlc338", "vlc301", "vlht45", "vlht04", "vlfw19", "vlfw12", "vl6m11", "vl6m03", "vlwv07", "vlwv40",
	"vl6833", "vl6824", "vlgo14", "vlgo46", "vluc10", "vluc24", "vl7d17", "vl7d01", "vlx642", "vlx608", "vlzq06", "vlzq41", "vlhp19", "vlhp01", "vl2o03", "vl2o13", "vlfn15", "vlfn20", "vl5r24",
	"vl5r10", "vlvm06", "vlvm18", "vlt047", "vlt041", "vlwn44", "vlwn06", "vlf748", "vlf727", "vlxr22", "vlxr03", "vll746", "vll706", "vlm017", "vlm049", "vlub19", "vlub01", "vl5v28", "vl5v04",
	"vlfs00", "vlfs11", "vl7u19", "vl7u20", "vlip15", "vlip10", "vl5s17", "vl5s19", "vlrh18", "vlrh39", "vl0l14", "vl0l20", "vl6o31", "vl6o35", "vl3v03", "vl3v21", "vl8926", "vl8938", "vlh629",
	"vlh646", "vljp21", "vljp49", "vlop37", "vlop31", "vll110", "vll114", "vllk16", "vllk24", "vlif47", "vlif39", "vlpy24", "vlpy33", "vl5c08", "vl5c30", "vlux35", "vlux45", "vlqm12", "vlqm30",
	"vlyd36", "vlyd24", "vl4r38", "vl4r24", "vl6p47", "vl6p07", "vls604", "vls644", "vlpx38", "vlpx11", "vldv04", "vldv36", "vlnj48", "vlnj49", "vl9q14", "vl9q44", "vlg405", "vlg427", "vl4u37",
	"vl4u20", "vldz01", "vldz26", "vl0315", "vl0319", "vlxz06", "vlxz45", "vly816", "vly820", "vl2g39", "vl2g23", "vlqu18", "vlqu19", "vlqw20", "vlqw25", "vlvu03", "vlvu21", "vlbk23", "vlbk22",
	"vl9b48", "vl9b36", "vl2844", "vl2827", "vlxj08", "vlxj41", "vl7e03", "vl7e17", "vlr830", "vlr846", "vlv213", "vlv219", "vlnz45", "vlnz11", "vldc17", "vldc41", "vl1i07", "vl1i22", "vlrz29",
	"vlrz14", "vlys06", "vlys48", "vley29", "vley11", "vlxe36", "vlxe44", "vll231", "vll246", "vlxx48", "vlxx17", "vlvc31", "vlvc06", "vla422", "vla400", "vlh806", "vlh810", "vlp507", "vlp538",
	"vlde47", "vlde32", "vlek15", "vlek37", "vlks14", "vlks39", "vlb507", "vlb501", "vldb44", "vldb09", "vlzt00", "vlzt37", "vlk905", "vlk943", "vl2x21", "vl2x03", "vlwc06", "vlwc30", "vldt48",
	"vldt25", "vldp10", "vldp48", "vlq230", "vlq215", "vlaz35", "vlaz43", "vlxc40", "vlxc48", "vlr234", "vlr213", "vl6a21", "vl6a30", "vl1b36", "vl1b38", "vlxh30", "vlxh42", "vlu803", "vlu812",
	"vl9x08", "vl9x47", "vlvy01", "vlvy46", "vl0t05", "vl0t06", "vlv735", "vlv749", "vlcu05", "vlcu43", "vlve15", "vlve39", "vlgc04", "vlgc16", "vl7k33", "vl7k26", "vlgv28", "vlgv26", "vlnu00",
	"vlnu46", "vlm822", "vlm815", "vl0a23", "vl0a04", "vlex14", "vlex22", "vlz142", "vlz132", "vl3g40", "vl3g49", "vlqx23", "vlqx24", "vlzv21", "vlzv31", "vlsh17", "vlsh29", "vlhb05", "vlhb32",
	"vlus10", "vlus21", "vlsi18", "vlsi43", "vl4p32", "vl4p30", "vl3w17", "vl3w46", "vly600", "vly628", "vlli04", "vlli27", "vlye24", "vlye33", "vl8333", "vl8314", "vlnn02", "vlnn06", "vlu306",
	"vlu330", "vl5g33", "vl5g00", "vl6b35", "vl6b24", "vlji21", "vlji42", "vlba17", "vlba12", "vls149", "vls132", "vl3c04", "vl3c07", "vlss28", "vlss18", "vluf46", "vluf15", "vl0025", "vl0033",
	"vlq131", "vlq100", "vlmd30", "vlmd14", "vlws10", "vlws49", "vl5u15", "vl5u04", "vlvr33", "vlvr00", "vlg929", "vlg915", "vlvb45", "vlvb19", "vl2t29", "vl2t36", "vl2z16", "vl2z03", "vle249",
	"vle221", "vlsd16", "vlsd09", "vl1m44", "vl1m21", "vltd04", "vltd21", "vl3u24", "vl3u16", "vlut31", "vlut44", "vl6q17", "vl6q23", "vl1x36", "vl1x11", "vlxd35", "vlxd44", "vlin17", "vlin20",
	"vl0k17", "vl0k23", "vlod36", "vlod04", "vl3221", "vl3238", "vlsp09", "vlsp34", "vlk501", "vlk529", "vl9s06", "vl9s28", "vlkf04", "vlkf42", "vlnq43", "vlnq36", "vl9j03", "vl9j01", "vlau34",
	"vlau42", "vl9p03", "vl9p46", "vlrn35", "vlrn03", "vli201", "vli212", "vluz02", "vluz01", "vlzm22", "vlzm49", "vl8f25", "vl8f45", "vljf18", "vljf32", "vlrj37", "vlrj28", "vlc033", "vlc044",
	"vlb410", "vlb433", "vlom28", "vlom01", "vl8700", "vl8737", "vlyp02", "vlyp01", "vl5844", "vl5803", "vlf834", "vlf823", "vlxn45", "vlxn31", "vl0y47", "vl0y17", "vlfj16", "vlfj31", "vl7p13",
	"vl7p39", "vlf137", "vlf111", "vlqd05", "vlqd48", "vlt129", "vlt133", "vl6235", "vl6249", "vljv46", "vljv26", "vlfl40", "vlfl30", "vlq027", "vlq030", "vlxq27", "vlxq44", "vl1g05", "vl1g39",
	"vlcd31", "vlcd20", "vlw601", "vlw634", "vlh946", "vlh934", "vl7f12", "vl7f19", "vlda12", "vlda30", "vl7943", "vl7942", "vlre46", "vlre31", "vlw026", "vlw030", "vlpm41", "vlpm20", "vlgl45",
	"vlgl19", "vlz638", "vlz632", "vleu31", "vleu26", "vlz749", "vlz745", "vlig15", "vlig11", "vl1d18", "vl1d00", "vlsm18", "vlsm38", "vli740", "vli713", "vltj19", "vltj05", "vlea14", "vlea03",
	"vlxo47", "vlxo07", "vl0436", "vl0416", "vllz09", "vllz31", "vlga29", "vlga19", "vlru31", "vlru25", "vl7531", "vl7515", "vlhi31", "vlhi10", "vl8i12", "vl8i26", "vl6z36", "vl6z07", "vlvf27",
	"vlvf40", "vlqq22", "vlqq16", "vlfb42", "vlfb06", "vltr07", "vltr09", "vlzi06", "vlzi03", "vljx41", "vljx45", "vlbd38", "vlbd34", "vlyh04", "vlyh11", "vlqs48", "vlqs06", "vl5y24", "vl5y06",
	"vlka47", "vlka08", "vlhg17", "vlhg35", "vlu019", "vlu024", "vlxy18", "vlxy11", "vl3125", "vl3127", "vl2r31", "vl2r10", "vl3f46", "vl3f42", "vloz35", "vloz07", "vlbh40", "vlbh42", "vlzy28",
	"vlzy44", "vlij22", "vlij09", "vlid13", "vlid29", "vl8c27", "vl8c44", "vlj336", "vlj344", "vlzs45", "vlzs31", "vlw804", "vlw844", "vlgp29", "vlgp43", "vl4322", "vl4308", "vlkl23", "vlkl40",
	"vlcc33", "vlcc07", "vl8z31", "vl8z38", "vlft42", "vlft14", "vlsa15", "vlsa28", "vlr046", "vlr043", "vljr37", "vljr47", "vlao26", "vlao20", "vl0q24", "vl0q17", "vljs00", "vljs34", "vlio23",
	"vlio34", "vlzo31", "vlzo34", "vlh142", "vlh110", "vltm01", "vltm27", "vlcz12", "vlcz40", "vlr315", "vlr325", "vl4v09", "vl4v30", "vll421", "vll436", "vla223", "vla205", "vlze30", "vlze00",
	"vlil42", "vlil08", "vl0f45", "vl0f00", "vl3720", "vl3744", "vllj01", "vllj04", "vl4h12", "vl4h36", "vlmo25", "vlmo10", "vl2n11", "vl2n14", "vllx22", "vllx16", "vl6704", "vl6724", "vl6d04",
	"vl6d41", "vleo40", "vleo42", "vlxw48", "vlxw09", "vlw728", "vlw700", "vl1l21", "vl1l19", "vl5133", "vl5121", "vlqa35", "vlqa23", "vlc635", "vlc611", "vlqf16", "vlqf21", "vlbu38", "vlbu44",
	"vl9a34", "vl9a14", "vlju47", "vlju33", "vlzr20", "vlzr45", "vlz845", "vlz824", "vlcw20", "vlcw24", "vl7618", "vl7602", "vlva41", "vlva10", "vlcr00", "vlcr02", "vlw130", "vlw128", "vlua16",
	"vlua31", "vl7h02", "vl7h29", "vlk317", "vlk302", "vlbf27", "vlbf42", "vlfa22", "vlfa08", "vleg33", "vleg10", "vlfc28", "vlfc19", "vlko03", "vlko43", "vly746", "vly700", "vlyg40", "vlyg42",
	"vll514", "vll546", "vl8t32", "vl8t16", "vl4z09", "vl4z07", "vlgr34", "vlgr14", "vly528", "vly544", "vlc217", "vlc238", "vlmw46", "vlmw32", "vldd22", "vldd06", "vlgy17", "vlgy19", "vl1w19",
	"vl1w31", "vlsu24", "vlsu29", "vlla29", "vlla39", "vln832", "vln830", "vlm218", "vlm240", "vl6120", "vl6141", "vlwo12", "vlwo37", "vloy38", "vloy43", "vlar38", "vlar09", "vlov42", "vlov07",
	"vlug29", "vlug02", "vlic44", "vlic38", "vlxv46", "vlxv00", "vlld33", "vlld20", "vli535", "vli545", "vlck47", "vlck06", "vld500", "vld516", "vlp007", "vlp024", "vlvz31", "vlvz28", "vllm40",
	"vllm02", "vl6445", "vl6413", "vlgf34", "vlgf06", "vl0m21", "vl0m35", "vlvk10", "vlvk02", "vlco24", "vlco34", "vlby05", "vlby46", "vlbm12", "vlbm39", "vl4t39", "vl4t17", "vly014", "vly032",
	"vljm08", "vljm24", "vl3s38", "vl3s11", "vl3j14", "vl3j38", "vlnx41", "vlnx09", "vldn39", "vldn24", "vl7i34", "vl7i26", "vlmu00", "vlmu15", "vl0h44", "vl0h07", "vlsj14", "vlsj38", "vlyb33",
	"vlyb00", "vldj23", "vldj47", "vlww19", "vlww03", "vlpq43", "vlpq42", "vl5e48", "vl5e27", "vlfm34", "vlfm26", "vla344", "vla313", "vl3442", "vl3431", "vlvs36", "vlvs42", "vlmy20", "vlmy31",
	"vlcm30", "vlcm48", "vlo103", "vlo131", "vlow00", "vlow04", "vlih05", "vlih10", "vld915", "vld933", "vlp801", "vlp827", "vlaq26", "vlaq17", "vlsg45", "vlsg01", "vloa05", "vloa22", "vl8u09",
	"vl8u49", "vlcx04", "vlcx32", "vl3515", "vl3537", "vlnl23", "vlnl24", "vld445", "vld437", "vld616", "vld629", "vliz41", "vliz25", "vlwe23", "vlwe15", "vldm32", "vldm19", "vl9401", "vl9400",
	"vl8205", "vl8240", "vle323", "vle343", "vldx37", "vldx29", "vlvx21", "vlvx04", "vlbo02", "vlbo37", "vl2h45", "vl2h07", "vlwi26", "vlwi32", "vloo40", "vloo29", "vlh024", "vlh042", "vlpl27",
	"vlpl22", "vl4x44", "vl4x28", "vlcv30", "vlcv09", "vlqe32", "vlqe37", "vl8e37", "vl8e38", "vl3304", "vl3319", "vllw34", "vllw12", "vlfv27", "vlfv47", "vlwz18", "vlwz36", "vlkd42", "vlkd49",
	"vl5505", "vl5526", "vlwj21", "vlwj06", "vl9y24", "vl9y46", "vljt25", "vljt36", "vlro12", "vlro05", "vlc849", "vlc807", "vl7v06", "vl7v17", "vlnp37", "vlnp45", "vldh12", "vldh09", "vlx049",
	"vlx043", "vl9f43", "vl9f39", "vl8r14", "vl8r31", "vlo811", "vlo835", "vlpp47", "vlpp20", "vlak42", "vlak18", "vljn39", "vljn49", "vlds09", "vlds37", "vlvg19", "vlvg05", "vlsq35", "vlsq45",
	"vlof48", "vlof46", "vlf333", "vlf348", "vlso14", "vlso09", "vle440", "vle403", "vl4c04", "vl4c12", "vlf433", "vlf408", "vlrb16", "vlrb28", "vl9u02", "vl9u24", "vlik39", "vlik18", "vlgn03",
	"vlgn44", "vlkh37", "vlkh26", "vlja23", "vlja47", "vl1744", "vl1708", "vlb730", "vlb735", "vlok15", "vlok29", "vl1p41", "vl1p27", "vltt39", "vltt31", "vlhe17", "vlhe48", "vlii01", "vlii06",
	"vlv339", "vlv322", "vls834", "vls823", "vltw46", "vltw11", "vlhq13", "vlhq18", "vl0p41", "vl0p29", "vlnr23", "vlnr15", "vljg32", "vljg21", "vl5o49", "vl5o21", "vl9z16", "vl9z39", "vlp748",
	"vlp725", "vl7240", "vl7212", "vl6y32", "vl6y10", "vl2205", "vl2231", "vlj543", "vlj508", "vlef15", "vlef17", "vldu20", "vldu11", "vlg838", "vlg835", "vlnw33", "vlnw08", "vlm945", "vlm928",
	"vl5n15", "vl5n27", "vl8o17", "vl8o07", "vldw31", "vldw14", "vlx843", "vlx801", "vlln44", "vlln47", "vl6319", "vl6316", "vlha13", "vlha06", "vlcg26", "vlcg43", "vlrr30", "vlrr23", "vlhc40",
	"vlhc05", "vlbp49", "vlbp17", "vljc00", "vljc27", "vl3647", "vl3606", "vlg515", "vlg500", "vlvd11", "vlvd20", "vlhj21", "vlhj27", "vltq43", "vltq41", "vl6l02", "vl6l46", "vln305", "vln317",
	"vltx15", "vltx06", "vl0x49", "vl0x43", "vlgk12", "vlgk36", "vls739", "vls720", "vl7t02", "vl7t31", "vlb609", "vlb608", "vlfu46", "vlfu22", "vlz523", "vlz503", "vlwp23", "vlwp49", "vl4o48",
	"vl4o18", "vlbv10", "vlbv43", "vl0n37", "vl0n18", "vl3y32", "vl3y16", "vlj637", "vlj617", "vl8k31", "vl8k30", "vlo649", "vlo620", "vlyv39", "vlyv02", "vld321", "vld343", "vlvp09", "vlvp34",
	"vlui44", "vlui30", "vlzf32", "vlzf43", "vlpj29", "vlpj43", "vl8q15", "vl8q25", "vl7818", "vl7843", "vlkt25", "vlkt46", "vlzp36", "vlzp41", "vljj04", "vljj45", "vl6s11", "vl6s49", "vlh312",
	"vlh307", "vlwh18", "vlwh38", "vl2504", "vl2501", "vl6t49", "vl6t11", "vlap20", "vlap14", "vlbj48", "vlbj27", "vlxu01", "vlxu23", "vlae37", "vlae20", "vlcp06", "vlcp16", "vlqk00", "vlqk07",
	"vl3829", "vl3834", "vlzx10", "vlzx14", "vlhl06", "vlhl24", "vl1810", "vl1804", "vlav02", "vlav19", "vleh44", "vleh20", "vl2p29", "vl2p41", "vlkg16", "vlkg05", "vl3l30", "vl3l22", "vlol42",
	"vlol48", "vly228", "vly216", "vl7436", "vl7430", "vl7w21", "vl7w15", "vlaf38", "vlaf27", "vlu629", "vlu645", "vlyk47", "vlyk25", "vlah28", "vlah21", "vlis14", "vlis24", "vlwa30", "vlwa04",
	"vlcb30", "vlcb26", "vl3n17", "vl3n20", "vl0b13", "vl0b25", "vlor18", "vlor05", "vl8n03", "vl8n42", "vl4n06", "vl4n49", "vlke33", "vlke04", "vlyz33", "vlyz44", "vlw407", "vlw410", "vlm104",
	"vlm113", "vl4l10", "vl4l14", "vlbq42", "vlbq10", "vla615", "vla608", "vlu104", "vlu138", "vlp938", "vlp902", "vlwr22", "vlwr00", "vlt334", "vlt327", "vl6g18", "vl6g49", "vl4a22", "vl4a24",
	"vljw14", "vljw30", "vl3z14", "vl3z02", "vl7024", "vl7028", "vlmh03", "vlmh36", "vl0u23", "vl0u31", "vlzh16", "vlzh31", "vlrp47", "vlrp05", "vlc936", "vlc921", "vlkn49", "vlkn30", "vlb909",
	"vlb940", "vlsf12", "vlsf37", "vljy25", "vljy03", "vlc507", "vlc542", "vlkp44", "vlkp37", "vlie02", "vlie04", "vlfo05", "vlfo22", "vle800", "vle841", "vl6w47", "vl6w20", "vl5l37", "vl5l42",
	"vltg44", "vltg25", "vlpe32", "vlpe35", "vl6k36", "vl6k40", "vlpn46", "vlpn34", "vlj018", "vlj007", "vlc407", "vlc417", "vlu206", "vlu222", "vl0w22", "vl0w37", "vlbx18", "vlbx31", "vli102",
	"vli121", "vlyn17", "vlyn33", "vlho16", "vlho09", "vllv34", "vllv47", "vl3047", "vl3044", "vlyt00", "vlyt12", "vlpf03", "vlpf29", "vlej02", "vlej18", "vl9w35", "vl9w36", "vl0s02", "vl0s31",
	"vlyq12", "vlyq16", "vlkc28", "vlkc07", "vlm409", "vlm443", "vlv530", "vlv537", "vltb34", "vltb01", "vl9l12", "vl9l32", "vlgu31", "vlgu48", "vlzw25", "vlzw35", "vlv844", "vlv826", "vlyy08",
	"vlyy18", "vl3d44", "vl3d14", "vl5k10", "vl5k41", "vlal06", "vlal42", "vlsc33", "vlsc49", "vld719", "vld718", "vl0g41", "vl0g38", "vlaw02", "vlaw47", "vln631", "vln619", "vlet22", "vlet03",
	"vlyl42", "vlyl22", "vlq424", "vlq409", "vlay26", "vlay21", "vlk743", "vlk729", "vlan41", "vlan29", "vl5h04", "vl5h23", "vl5d14", "vl5d47", "vl8x28", "vl8x19", "vlg718", "vlg709", "vluh11",
	"vluh03", "vly941", "vly931", "vl6610", "vl6608", "vl5a44", "vl5a41", "vlrw21", "vlrw01", "vle949", "vle945", "vlt630", "vlt642", "vl5q41", "vl5q20", "vlz923", "vlz905", "vl7l27", "vl7l41",
	"vl4y04", "vl4y33", "vlmf14", "vlmf10", "vl8538", "vl8502", "vlcf35", "vlcf23", "vldo18", "vldo26", "vlx512", "vlx534", "vl7a35", "vl7a32", "vl9v42", "vl9v10", "vl7m41", "vl7m07", "vlsl00",
	"vlsl47", "vlta05", "vlta16", "vlgd04", "vlgd15", "vlqz32", "vlqz12", "vlku41", "vlku46", "vl7q40", "vl7q01", "vlgt34", "vlgt25", "vlj110", "vlj138", "vlq911", "vlq913", "vlnd23", "vlnd17",
	"vlzb27", "vlzb48", "vlzn09", "vlzn16", "vlem34", "vlem07", "vl3a25", "vl3a46", "vlkw13", "vlkw31", "vle635", "vle607", "vlbs38", "vlbs44", "vlxp02", "vlxp19", "vlxb44", "vlxb18", "vlxi23",
	"vlxi06", "vlt512", "vlt519", "vl5225", "vl5205", "vldi14", "vldi23", "vl8g20", "vl8g33", "vlni04", "vlni33", "vlg601", "vlg646", "vlfr26", "vlfr25", "vluq05", "vluq27", "vl3r46", "vl3r14",
	"vl7r48", "vl7r26", "vlq347", "vlq341", "vlk808", "vlk833", "vl6v21", "vl6v45", "vlw346", "vlw307", "vlz209", "vlz201", "vlmi43", "vlmi25", "vl1q02", "vl1q39", "vlai10", "vlai48", "vlew03",
	"vlew21", "vlbz08", "vlbz32", "vl1z30", "vl1z32", "vlib27", "vlib11", "vln024", "vln035", "vl1521", "vl1545", "vl4g04", "vl4g22", "vlse41", "vlse15", "vlbe32", "vlbe28", "vl7g31", "vl7g33",
	"vlir24", "vlir16", "vloq37", "vloq10", "vlnk45", "vlnk39", "vlf522", "vlf545", "vlb123", "vlb141", "vl1204", "vl1202", "vlqt38", "vlqt32", "vl6e35", "vl6e49", "vl1k38", "vl1k13", "vl9h02",
	"vl9h38", "vlk615", "vlk637", "vln723", "vln733",
}
