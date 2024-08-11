
var questionResponse = convertResponse(JSON.parse(response.content));

context.setVariable("response.content", JSON.stringify({
  "answer": questionResponse
}));

function convertResponse(dataResponseObject) {
  var result = "";

  for (i = 0; i < dataResponseObject.length; i++) {
    result += dataResponseObject[i]["candidates"][0]["content"]["parts"][0]["text"];
  }

  return result;
}

// this is to only export the function if in node
if (typeof exports !== 'undefined') {
  exports.convertResponse = convertResponse;
}