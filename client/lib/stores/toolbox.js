var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')

var chat = require('./chat')
var socket = require('./socket')


var storeActions = Reflux.createActions([
  'chooseCommand',
  'apply',
])
_.extend(module.exports, storeActions)

var StateRecord = Immutable.Record({
  items: Immutable.Set(),
  selectedCommand: 'delete',
  activeItemSummary: 'nothing',
})

var commands = {
  delete: {
    kind: 'message',
    execute: function(items) {
      items.forEach(item => {
        socket.send({
          type: 'edit-message',
          data: {
            id: item.get('id'),
            delete: true,
            announce: true,
          },
        })
      })
    }
  },
  ban: {
    kind: 'user',
    execute: function(items, commandParams) {
      items.forEach(item => {
        socket.send({
          type: 'ban',
          data: {
            id: item.get('id'),
            seconds: commandParams.seconds,
          },
        })
      })
    }
  },
}

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {chatUpdate: chat.store},
    {messagesUpdate: chat.messagesChanged},
  ],

  mixins: [require('./immutable-mixin')],

  init: function() {
    this.state = new StateRecord()
  },

  getInitialState: function() {
    return this.state
  },

  chatUpdate: function(chatState) {
    this.triggerUpdate(this._updateSelection(this.state, chatState))
  },

  messagesUpdate: function(ids, chatState) {
    this.triggerUpdate(this._updateSelection(this.state, chatState))
  },

  _updateSelection: function(state, chatState) {
    if (chatState.selectedMessages.size) {
      state = state.set('items',
        chatState.selectedMessages
          .toSeq()
          .map(id => {
            var message = chatState.messages.get(id)
            if (!message || !message.get('$count')) {
              return
            }

            var sender = message.get('sender')
            var senderId = sender.get('id')
            return Immutable.fromJS([
              {
                kind: 'message',
                id: id,
                removed: !!message.get('deleted'),
              },
              {
                kind: 'user',
                id: senderId,
                name: sender.get('name'),
                removed: chatState.bannedIds.has(senderId),
              },
            ])
          })
          .filter(Boolean)
          .flatten(1)
          .toSet()
          .sortBy(item => [!item.get('removed'), item.get('kind')])
      )
      state = this._updateFilter(state)
    } else {
      state = state.delete('items')
      state = state.delete('activeItemSummary')
    }
    return state
  },

  _updateFilter: function(state) {
    var commandKind = commands[state.selectedCommand].kind

    state = state.set('items',
      state.items.map(
        item => item.set('active', !item.get('removed') && item.get('kind') == commandKind)
      )
    )

    var activeCount = state.items.count(item => item.get('active'))

    if (activeCount) {
      // TODO: tricky localization
      state = state.set('activeItemSummary', activeCount + ' ' + commandKind + (activeCount == 1 ? '' : 's'))
    } else {
      state = state.set('activeItemSummary', 'nothing')
    }

    return state
  },

  chooseCommand: function(command) {
    var state = this.state.set('selectedCommand', command)
    this.triggerUpdate(this._updateFilter(state))
  },

  apply: function(commandParams) {
    var activeItems = this.state.items.filter(item => item.get('active'))
    commands[this.state.selectedCommand].execute(activeItems, commandParams)
  },
})
