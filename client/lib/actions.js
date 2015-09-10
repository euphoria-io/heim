var Reflux = require('reflux')

module.exports = Reflux.createActions([
  'sendMessage',
  'loadMoreLogs',
  'setNick',
  'tryRoomPasscode',
  'connect',
  'joinRoom',
  'embedMessage',
])

// sync so that we connect and set joinWhenReady in the load tick
module.exports.connect.sync = true
module.exports.joinRoom.sync = true

// sync so that chatEntry can pass its state off to tentativeNick immediately after calling setNick
module.exports.setNick.sync = true

// sync so that embed components can react quickly to events
module.exports.embedMessage.sync = true
