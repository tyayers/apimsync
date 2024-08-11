var bqResponse = context.getVariable("response.content");
var pageSize = context.getVariable("bq.pageSize");
var pageToken = context.getVariable("bq.pageToken");
var entityName = context.getVariable("entityName");

var responseObject = convertResponse(JSON.parse(bqResponse), entityName, pageToken);

context.setVariable("response.content", JSON.stringify(responseObject));
