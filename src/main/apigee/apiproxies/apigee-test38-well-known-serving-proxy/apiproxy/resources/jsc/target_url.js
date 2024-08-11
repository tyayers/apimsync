// Get GCS bucket storing the plugin manifest.
var bucket = context.getVariable('chatgpt_plugin_bucket');

// Get the folder storing manifeset of this plugin.
var folder = context.getVariable('chatgt_plugin_name');

// Get the resource file path
var pathSuffix = context.getVariable('proxy.pathsuffix');

// Construct the final target url to GCS
var gcsBase = 'https://storage.googleapis.com';
var targetUrl = folder ? (gcsBase + '/' + bucket + '/' + folder + pathSuffix) : 'not-found';

context.setVariable('target.url', targetUrl);

