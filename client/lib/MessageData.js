import Immutable from 'immutable'
import EventEmitter from 'eventemitter3'


export default class MessageData {
  constructor(initMessageData) {
    this.initMessageData = Immutable.fromJS(initMessageData)
    this.data = {}
    this.changes = new EventEmitter()
  }

  get(messageId) {
    return this.data[messageId] || this.initMessageData
  }

  set(messageId, data) {
    const messageData = this.data[messageId] || this.initMessageData
    const newMessageData = this.data[messageId] = messageData.merge(data)
    this.changes.emit(messageId, newMessageData)
  }
}
