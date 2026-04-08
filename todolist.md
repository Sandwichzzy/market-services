
1. 法币的业务完成
- 完成法币API读取入库：
只接入ExchangeRate-API， 需要构建yaml配置 conf.APIKeyConfig.ExchangeRate,
- 然后需要构建HTTP和grpc服务

2. 从 CMC 获取数据完善
Volume: 交易量
MarketCap: 市值
https://coinmarketcap.com/ 支持 API 调用的，可以去找一下 API

3. 完成交易对市场价格的restapi和grpc服务

