
1. 法币的数据处理入库完成（完成）
完成从法币API读取入库currency
目前只接入ExchangeRate-API， 需要构建yaml配置 conf.APIKeyConfig.ExchangeRate
如何读取yaml配置到config中


2. 从 CMC 获取数据完善
Volume: 交易量
MarketCap: 市值
https://coinmarketcap.com/ 支持 API 调用的，可以去找一下 API

3. 完成对外接口
完成market-symbol价格的restapi和grpc服务
然后法币的业务:构建restapi和grpc服务
暂时不做kline相关业务

