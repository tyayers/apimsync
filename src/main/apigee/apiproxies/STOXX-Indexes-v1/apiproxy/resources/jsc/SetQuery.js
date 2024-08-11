context.targetRequest.method='POST';
context.targetRequest.headers['Content-Type']='application/json';

context.setVariable("target.copy.pathsuffix", false);

var filter = "";
var orderBy = "";
var pageSize = "";
var pageToken = "";

for(var queryParam in request.queryParams){
    if (queryParam == "filter") {
        filter = "WHERE " + context.getVariable("request.queryparam." + queryParam);
    }
    else if (queryParam == "orderBy") {
        orderBy = "ORDER BY " + context.getVariable("request.queryparam." + queryParam);
    }
    else if (queryParam == "pageSize") {
        var tempPageSize =  context.getVariable("request.queryparam." + queryParam);
        pageSize = "LIMIT " + tempPageSize;
        context.setVariable("bq.pageSize", tempPageSize);
    }
    else if (queryParam == "pageToken") {
        var tempPageToken =  context.getVariable("request.queryparam." + queryParam);
        pageToken = "OFFSET " + parseInt(context.getVariable("request.queryparam.pageSize")) * (parseInt(tempPageToken) - 1);
        context.setVariable("bq.pageToken", tempPageToken);
    }
}

if (pageSize == ""){
  // Set default pageSize to 10
  pageSize = "LIMIT 10";
  context.setVariable("bq.pageSize", pageSize);
}

var query = "";
var table = "test.index_research";

if (table)
  query = "SELECT * FROM " + table + " %filter% %orderBy% %pageSize% %pageToken%";

query = query.replace("%filter%", filter);
query = query.replace("%orderBy%", orderBy);
query = query.replace("%pageSize%", pageSize);
query = query.replace("%pageToken%", pageToken);

context.targetRequest.body = '' +
    '{' + 
    '   "query": "' + query + '",' +            
    '   "useLegacySql": false,' +
    '   "maxResults": 1000' +
    '}';