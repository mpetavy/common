// run `node index.js` in the terminal
const HL7 = require('hl7-standard');
const MOMENT = require('moment');

var hl

var date = new Date();
var formattedDate = MOMENT(date).format('YYYYMMDD');

console.log("!!! ",formattedDate)

// console.log(`Hello Node.js v${process.versions.node}!`);

let message =
  new HL7(`MSH|^~\&|EPIC|EPICADT|SMS|SMSADT|199912271408|CHARRIS|ADT^A04|1817457|D|2.5|\n
PID||0493575^^^2^ID 1|454721||DOE^JOHN^^^^|DOE^JOHN^^^^|19480203|M||B|254 MYSTREET AVE^^MYTOWN^OH^44123^USA||(216)123-4567|||M|NON|400003403~1129086|\n
NK1||ROE^MARIE^^^^|SPO||(216)123-4567||EC|||||||||||||||||||||||||||\n
PV1||O|168 ~219~C~PMA^^^^^^^^^||||277^ALLEN MYLASTNAME^BONNIE^^^^|||||||||| ||2688684|||||||||||||||||||||||||199912271408||||||002376853\n
`);

// callback optional
try {
  // code here
  message.transform((err) => {
    if (err) throw err;

    console.log(JSON.stringify(message.transformed));

    for (let segment of message.getSegments()) {
      // loops through all segments in the message
    }

    for (let obr of message.getSegments('OBR')) {
      // loops through only the OBR segments in the message
    }

    for (let [i, segment] of message.getSegments().entries()) {
      // for of loop with index of iteration
    }

    let segments = message.getSegments();
    for (var i = 0; i < segments.length; i++) {
      // using a standard for loop
    }
  });
} catch (e) {
  console.error(e);
}

message.build();
