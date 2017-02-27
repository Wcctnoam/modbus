var RED = require(process.env.NODE_RED_HOME+"/red/red");
var Go = require('gonode').Go;
//var configFile = "/root/work/src/github.com/foxconn4tech/modbus/main/conf.json"

module.exports = function(RED) {

	function  ModbusServer(n){
		RED.nodes.createNode(this, n)
		this.status({fill:"green",shape:"dot",text:"Started"});
		var go = new Go({path: 'main/main_v2.go', initAtOnce: true}, function(err) {
			if(err) {
				console.log('Error initializing.\n' + err);
				return;
			}
			go.on('error', function(err) {
				console.log('Error from gonode.\n'
					+ 'Is it from the internal parser?' + (err.parser ? 'yes' : 'no')
					+ '\nActual error: \n'
					+ err.data.toString());
			});


			var inputjson = {

				mqttaddr: n.mqttaddr,
				mqtttport: n.mqttport,
				modbusClientaddr: n.modbusClientadd,
				modbusClientport: n.modbusClientport

			};
			var check;

			check = go.execute(inputjson, function(result, response) {
				console.log("1" );
				console.log("OK: " + result.ok);
				console.log("Timeout: " + result.timeout);
				console.log("Terminated: " + result.terminated);
				if(result.ok) {
					console.log('Go finished correctly.')
					console.log('Go responded: ' + response.text + '\n\n');
				} else {
					console.log('Something went wrong with the \'Connect\' command.');
					console.log('Go responded: ' + response.text + '\n\n');
				}
			});

			go.close();
		});
	}
	RED.nodes.registerType("ModbusServer", ModbusServer);

}
