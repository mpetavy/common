const HL7 = require('hl7-standard');
const MOMENT = require('moment');

var hl7 = new HL7();
var date = new Date();
var timestamp = MOMENT(date).format('YYYYMMDD');

hl7.createSegment('MSH');
hl7.set('MSH', {
    'MSH.2': '^~\\&',
    'MSH.3': 'Example',
    'MSH.4': '123456',
    'MSH.5': '',
    'MSH.6': '',
    'MSH.7': timestamp,
    'MSH.8': '',
    'MSH.9': {
        'MSH.9.1': 'ADT',
        'MSH.9.2': 'A08'
    },
    'MSH.10': '',
    'MSH.11': 'T',
    'MSH.12': '2.3'
});

hl7.build();
