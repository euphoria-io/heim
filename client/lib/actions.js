var Reflux = require('reflux')

module.exports = Reflux.createActions([
  'sendMessage',
  'focusMessage',
  'toggleFocusMessage',
  'setEntryText',
  'focusEntry',
  'scrollToEntry',
  'keydownOnEntry',
  'loadMoreLogs',
  'showSettings',
  'setNick',
  'tryRoomPasscode',
  'connect',
])

// sync so that we connect in the load tick
module.exports.connect.sync = true

// sync so that chatEntry can pass its state off to tentativeNick immediately after calling setNick
module.exports.setNick.sync = true

// sync to allow entry to preventDefault keydown events
module.exports.keydownOnEntry.sync = true
