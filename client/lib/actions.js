var Reflux = require('reflux')

module.exports = Reflux.createActions([
  'sendMessage',
  'loadMoreLogs',
  'setNick',
  'tryRoomPasscode',
  'setup',
  'connect',
  'joinRoom',
  'embedMessage',
])

// sync so that we initialize room name / storage in the load tick
module.exports.setup.sync = true

// sync so that chatEntry can pass its state off to tentativeNick immediately after calling setNick
module.exports.setNick.sync = true

// sync so that embed components can react quickly to events
module.exports.embedMessage.sync = true
