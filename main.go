package main

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/lwzphper/cart-master/common"
	"github.com/lwzphper/cart-master/domain/repository"
	service2 "github.com/lwzphper/cart-master/domain/service"
	"github.com/lwzphper/cart-master/handler"
	cart "github.com/lwzphper/cart-master/proto"
	"github.com/micro/go-micro/v2"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	consul2 "github.com/micro/go-plugins/registry/consul/v2"
	ratelimit "github.com/micro/go-plugins/wrapper/ratelimiter/uber/v2"
	opentracing2 "github.com/micro/go-plugins/wrapper/trace/opentracing/v2"
	"github.com/opentracing/opentracing-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var QPS = 100

func main() {
	// 配置中心
	consulConfig, err := common.GetConsulConfig("192.168.110.20", 8500, "/micro/config")
	if err != nil {
		log.Error(err)
	}
	// 注册中心
	consul := consul2.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{
			"192.168.110.20:8500",
		}
	})

	// 链路追踪
	t, io, err := common.NewTracer("go.micro.service.cart", "192.168.110.20:6831")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)

	// 数据库设置。 consul 的目录路径：micro/config/mysql
	mysqlInfo := common.GetMysqlFromConsul(consulConfig, "mysql")
	db, err := gorm.Open(mysql.Open(mysqlInfo.User+":"+mysqlInfo.Pwd+"@/"+mysqlInfo.Database+"?charset=utf8&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		log.Error(err)
	}

	// 初始化表
	err = repository.NewCartRepository(db).InitTable()
	if err != nil {
		log.Error(err)
	}

	// New Service
	service := micro.NewService(
		micro.Name("go.micro.service.cart"),
		micro.Version("latest"),
		// 暴露服务地址
		micro.Address("0.0.0.0:8087"),
		// 注册中心
		micro.Registry(consul),
		// 链路追踪
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
		// 添加限流
		micro.WrapHandler(ratelimit.NewHandlerWrapper(QPS)),
		)

	// Initialize service
	service.Init()

	cartDataService := service2.NewCartDataService(repository.NewCartRepository(db))

	// Register Handler
	cart.RegisterCartHandler(service.Server(), &handler.Cart{CartDataService: cartDataService})

	// Run Service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}