var _ = require('lodash')
var Immutable = require('immutable')
var EventEmitter = require('eventemitter3')


function MessageData(initMessageData) {
  this.initMessageData = Immutable.fromJS(initMessageData)
  this.data = {}
  this.changes = new EventEmitter()
}

_.extend(MessageData.prototype, {
  get: function(messageId) {
    return this.data[messageId] || this.initMessageData
  },

  set: function(messageId, data) {
    var messageData = this.data[messageId] || this.initMessageData
    messageData = this.data[messageId] = messageData.merge(data)
    this.changes.emit(messageId, messageData)
  },
})

module.exports = MessageData
