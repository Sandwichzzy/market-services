
1. 法币的数据处理入库完成
完成从法币API读取入库currency
目前只接入ExchangeRate-API
需要构建yaml配置 conf.APIKeyConfig.ExchangeRate
如何读取yaml配置到config中


2. 从 CMC 获取数据完善
Volume: 交易量
MarketCap: 市值
https://coinmarketcap.com/ 支持 API 调用的，可以去找一下 API

APIKEY:, 配置到环境变量或者yaml文件
使用方式：
Call https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest.
Pass symbol=BTC and convert=USD in the query string.
Read market_cap and volume_24h from the quote.USD object in the response.
You should use HTTPS GET requests with the header X-CMC_PRO_API_KEY: .

Example Request And Fields
Example HTTP request (conceptually):

Method: GET
URL:
https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest?symbol=BTC&amp;convert=USD
Headers:
X-CMC_PRO_API_KEY: YOUR_API_KEY
Accept: application/json
In many languages or tools (curl, Python requests, Node fetch, etc) you just set that URL and header.

Response shape (simplified):

The JSON will look like:

data.BTC[0].quote.USD.market_cap
data.BTC[0].quote.USD.volume_24h

Those two fields are:

market_cap
Current Bitcoin market capitalization in USD.

volume_24h
Total traded volume in the last 24 hours in USD.

Using id instead of symbol (optional, more robust):

First, get Bitcoin’s id from GET /v1/cryptocurrency/map or from your own metadata.
Then call:
https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest?id=&amp;convert=USD
Read:
data[""].quote.USD.market_cap
data[""].quote.USD.volume_24h


3. 完成对外接口
完成market-symbol价格的restapi和grpc服务
然后法币的业务:构建restapi和grpc服务


