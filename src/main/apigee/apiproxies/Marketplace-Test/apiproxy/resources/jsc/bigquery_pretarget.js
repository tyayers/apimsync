context.targetRequest.method='POST';
context.targetRequest.headers['Content-Type']='application/json';

var entityName = context.getVariable("entityName");
var objectName = context.getVariable(entityName);
if (!objectName) objectName = entityName;

print("BigQuery entity name: " + entityName + "\n");
print("BigQuery object name: " + objectName + "\n");

var query = "";
var table = "";
if (objectName.startsWith("table::"))
  table = objectName.replace("table::", "");
else if (objectName.startsWith("query::"))
  query = objectName.replace("query::", "");
else
  table = objectName;

print("BigQuery table input: " + table + "\n");
print("BigQuery query input: " + query + "\n");
context.setVariable("target.copy.pathsuffix", false);

var filter = "";
var orderBy = "";
var pageSize = "";
var pageToken = "";

for(var queryParam in request.queryParams){
  if (queryParam == "filter") {
      filter = context.getVariable("request.queryparam." + queryParam);
  }
  else if (queryParam == "orderBy") {
      orderBy = context.getVariable("request.queryparam." + queryParam);
  }
  else if (queryParam == "pageSize") {
      var pageSize =  context.getVariable("request.queryparam." + queryParam);
      context.setVariable("bq.pageSize", pageSize);
  }
  else if (queryParam == "pageToken") {
      pageToken =  context.getVariable("request.queryparam." + queryParam);
      context.setVariable("bq.pageToken", pageToken);
  }
}

var newQuery = generateQuery(query, table, filter, orderBy, pageSize, pageToken);

print("BigQuery query: " + newQuery);

context.targetRequest.body = '' +
  '{' + 
  '   "query": "' + newQuery + '",' +            
  '   "useLegacySql": false,' +
  '   "maxResults": 1000' +
  '}';