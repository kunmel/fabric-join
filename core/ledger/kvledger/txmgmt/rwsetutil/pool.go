package rwsetutil

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/hyperledger/fabric/protos/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrNotFoundClient = errors.New("not found grpc conn")
	ErrConnShutdown   = errors.New("grpc conn shutdown")

	defaultClientPoolConnsSizeCap    = 2000
	defaultDialTimeout      = 5 * time.Second
	defaultKeepAlive        = 30 * time.Second
	defaultKeepAliveTimeout = 10 * time.Second
)

type ClientOption struct {
	ClientPoolConnsSize  	int
	DialTimeOut				time.Duration
	KeepAlive				time.Duration
	KeepAliveTimeout		time.Duration
}


type ClientPool struct {
	target 		string
	option 		*ClientOption
	next 		int64
	cap 		int64

	sync.Mutex

	conns 		[]*grpc.ClientConn
}

func (cc *ClientPool) getConn() (*grpc.ClientConn, error){
	var (
		idx 		int64
		next		int64
		err			error
	)

	next = atomic.AddInt64(&cc.next, 1)
	//
	idx = next % cc.cap
	fmt.Println("USING CONN INDEX ", idx)
	conn := cc.conns[idx]
	if conn != nil && cc.checkState(conn) == nil {
		return conn, nil
	}

	//gc old conn
	if conn != nil {
		conn.Close()
	}

	cc.Lock()
	defer cc.Unlock()

	//double check, Prevent have been initialized
	if conn != nil && cc.checkState(conn) == nil {
		return conn, nil
	}

	conn, err = cc.connect()
	if err != nil {
		return nil, err
	}

	cc.conns[idx] = conn
	return conn, nil
}

func (cc *ClientPool) checkState(conn *grpc.ClientConn) error {
	state := conn.GetState()
	switch state {
	case connectivity.TransientFailure, connectivity.Shutdown:
		return ErrConnShutdown
	}

	return nil
}

func (cc *ClientPool) connect() (*grpc.ClientConn, error) {
	ctx, cal := context.WithTimeout(context.TODO(), cc.option.DialTimeOut)
	defer cal()
	conn, err := grpc.DialContext(ctx,
		cc.target,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:		cc.option.KeepAlive,
			Timeout:	cc.option.KeepAliveTimeout,
		}),
		grpc.WithInitialWindowSize(1<<30),
		grpc.WithInitialConnWindowSize(1<<30))
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// close all conn in pool
func (cc *ClientPool) Close() {
	cc.Lock()
	defer cc.Unlock()

	for _, conn := range cc.conns {
		if conn == nil {
			continue
		}

		conn.Close()
	}
}

// create new empty pool
func NewClientPoolWithOption(target string, option *ClientOption) *ClientPool {
	if (option.ClientPoolConnsSize) <= 0 {
		option.ClientPoolConnsSize = defaultClientPoolConnsSizeCap
	}

	if option.DialTimeOut <= 0 {
		option.DialTimeOut = defaultDialTimeout
	}

	if option.KeepAlive <= 0 {
		option.KeepAlive = defaultKeepAlive
	}

	if option.KeepAliveTimeout <= 0 {
		option.KeepAliveTimeout = defaultKeepAliveTimeout
	}


	return &ClientPool{
		target: target,
		option: option,
		cap:    int64(option.ClientPoolConnsSize),
		conns:   make([]*grpc.ClientConn, option.ClientPoolConnsSize)	,
	}
}

type TargetServiceNames struct {
	m map[string][]string
}

func NewTargetServiceNames() *TargetServiceNames {
	return &TargetServiceNames{
		m: make(map[string][]string),
	}
}

func (h *TargetServiceNames) Set(target string, serviceNames ...string) {
	if len(serviceNames) <= 0 {
		return
	}

	soureServNames := h.m[target]
	for _, sn := range serviceNames {
		soureServNames = append(soureServNames, sn)
	}

	h.m[target] = soureServNames
}

func (h *TargetServiceNames) list() map[string][]string {
	return h.m
}

func (h *TargetServiceNames) len() int {
	return len(h.m)
}

//通过属性clients以服务名为key去map里取ClientPool连接池里的clientconn
type ServiceClientPool struct {
	clients map[string]*ClientPool
	option *ClientOption
	clientCap int
}

func NewServiceClientPool(option *ClientOption) *ServiceClientPool {
	return &ServiceClientPool{
		option:    option,
		clientCap: option.ClientPoolConnsSize,
	}
}

func (sc *ServiceClientPool) Init(m *TargetServiceNames) {

	var clients = make(map[string]*ClientPool, m.len())

	for target, servNameArr := range m.list() {
		cc := NewClientPoolWithOption(target, sc.option)
		for _, srv := range servNameArr {
			clients[srv] = cc
		}
	}

	sc.clients =  clients
}

func (sc *ServiceClientPool) GetClientWithFullMethod(fullMethod string) (*grpc.ClientConn, error){
	sn := sc.SpiltFullMethod(fullMethod)
	return sc.GetClient(sn)
}

func (sc *ServiceClientPool) GetClient(sname string) (*grpc.ClientConn, error) {
	cc, ok := sc.clients[sname]
	if !ok {
		return nil, ErrNotFoundClient
	}

	return cc.getConn()
}

func (sc *ServiceClientPool) Close(sname string) {
	cc, ok := sc.clients[sname]
	if !ok {
		return
	}

	cc.Close()
}

func (sc *ServiceClientPool) CloseAll() {
	for _, client := range sc.clients {
		client.Close()
	}
}

// split 47 return the second string (the first one is "")
func (sc *ServiceClientPool) SpiltFullMethod(fullMethod string) string {
	var arr []string

	arr = strings.Split(fullMethod, "/")
	if len(arr) != 3 {
		return ""
	}

	return arr[1]
}

func (sc *ServiceClientPool) Invoke(ctx context.Context, fullMethod string, args *pb.Response, opts ...grpc.CallOption) (pb.SimulateResult, error) {
	//var md metadata.MD
	var sgxResult pb.SimulateResult
	sname := sc.SpiltFullMethod(fullMethod)
	conn, err := sc.GetClient(sname)
	if err != nil {
		return sgxResult,err
	}

	//md, flag := metadata.FromOutgoingContext(ctx)
	//if flag == true {
	//	md = md.Copy()
	//} else {
	//	md = metadata.MD{}
	//}
	//
	//for k, v := range headers {
	//	md.Set(k, v)
	//}
	//
	//ctx = metadata.NewOutgoingContext(ctx, md)
	client := pb.NewSgxServiceClient(conn)
	serverStream, err := client.NewSgxTx(context.Background(), &pb.NewTxRequest{
		OffChainData: args.OffChainData,
		WorkLoad: args.WorkLoad,
		OnChainData: getOnChainData(args),
		TxID: args.TXID,
	})
	if err != nil {
		log.Fatalln(err)
	}

	for {
		result, err := serverStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		// 此处的复制要放在err的判断后
		sgxResult.TxID = result.TxID
		sgxResult.WriteSet = result.WriteSet
	}
	return sgxResult,nil
	//return conn.Invoke(ctx, fullMethod, args, reply, opts...)
}

const (
	ADDRESS = "39.107.98.10:1234"
	//ADDRESS = "192.168.1.106:1234"
	SERVICENAME = "hello.sgxTx"
)

var scp *ServiceClientPool

func init() {
	co := ClientOption{
		ClientPoolConnsSize: defaultClientPoolConnsSizeCap,
		DialTimeOut:         defaultDialTimeout,
		KeepAlive:           defaultKeepAlive,
		KeepAliveTimeout:    defaultKeepAliveTimeout,
	}
	scp = NewServiceClientPool(&co)
	tsn := NewTargetServiceNames()
	tsn.Set(ADDRESS, SERVICENAME)
	scp.Init(tsn)
}

func GetScp() *ServiceClientPool {
	return scp
}

func getOnChainData(res *pb.Response) ([]*pb.DataMap){
	onChainData := Split47(string(res.Payload))
	half := len(onChainData)/2
	dataMap := make([]*pb.DataMap, half)
	//var dataMap  []*pb.DataMap
	for k := 0; k < half; k++ {
		var dataMap2  pb.DataMap
		dataMap2.Key = onChainData[k]
		dataMap2.Value = onChainData[k+half]
		dataMap[k] = &dataMap2
	}
	return dataMap
}
