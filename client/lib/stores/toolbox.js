import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import chat from './chat'
import ImmutableMixin from './ImmutableMixin'


const storeActions = Reflux.createActions([
  'chooseCommand',
  'apply',
])
_.extend(module.exports, storeActions)

const StateRecord = Immutable.Record({
  items: Immutable.Set(),
  selectedCommand: 'delete',
  activeItemSummary: 'nothing',
})

const commands = {
  delete: {
    kind: 'message',
    execute(items) {
      items.forEach(item =>
        chat.editMessage(item.get('id'), {
          delete: true,
          announce: true,
        })
      )
    },
  },
  ban: {
    kind: 'user',
    execute(items, commandParams) {
      items.forEach(item =>
        chat.banUser(item.get('id'), {
          seconds: commandParams.seconds,
        })
      )
    },
  },
  banIP: {
    kind: 'user',
    filter(item) {
      return !!item.get('ipAddr')
    },
    execute(items, commandParams) {
      items.forEach(item =>
        chat.banIP(item.get('ipAddr'), {
          seconds: commandParams.seconds,
          global: commandParams.global,
        })
      )
    },
  },
}

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {chatUpdate: chat.store},
    {messagesUpdate: chat.messagesChanged},
  ],

  mixins: [ImmutableMixin],

  init() {
    this.state = new StateRecord()
  },

  getInitialState() {
    return this.state
  },

  chatUpdate(chatState) {
    this.triggerUpdate(this._updateSelection(this.state, chatState))
  },

  messagesUpdate(ids, chatState) {
    this.triggerUpdate(this._updateSelection(this.state, chatState))
  },

  _updateSelection(startState, chatState) {
    let state = startState

    const messageItems = chatState.selectedMessages
      .toSeq()
      .map(id => {
        const message = chatState.messages.get(id)
        if (!message || !message.get('$count')) {
          return false
        }

        const sender = message.get('sender')
        const senderId = sender.get('id')
        const userInfo = chatState.who.get(sender.get('session_id'))
        const ipAddr = userInfo && userInfo.get('client_address')
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
            ipAddr,
            removed: chatState.bannedIds.has(senderId) || chatState.bannedIPs.has(ipAddr),
          },
        ])
      })
      .filter(Boolean)
      .flatten(1)
      .toSet()

    const userItems = chatState.selectedUsers
      .map(sessionId => {
        const userInfo = chatState.who.get(sessionId)
        if (!userInfo) {
          return false
        }

        const userId = userInfo.get('id')
        const ipAddr = userInfo.get('client_address')
        return Immutable.Map({
          kind: 'user',
          id: userId,
          name: userInfo.get('name'),
          ipAddr,
          removed: chatState.bannedIds.has(userId) || chatState.bannedIPs.has(ipAddr),
        })
      })
      .filter(Boolean)

    if (messageItems.size || userItems.size) {
      state = state.set('items',
        messageItems
          .union(userItems)
          .sortBy(item => [!item.get('removed'), item.get('kind')])
      )
      state = this._updateFilter(state)
    } else {
      state = state.delete('items')
      state = state.delete('activeItemSummary')
    }
    return state
  },

  _updateFilter(startState) {
    let state = startState
    const commandKind = commands[state.selectedCommand].kind
    const commandFilter = commands[state.selectedCommand].filter || (() => true)

    state = state.set('items',
      state.items.map(
        item => item.set('active', !item.get('removed') && item.get('kind') === commandKind && commandFilter(item))
      )
    )

    const activeCount = state.items.count(item => item.get('active'))

    if (activeCount) {
      // TODO: tricky localization
      state = state.set('activeItemSummary', activeCount + ' ' + commandKind + (activeCount === 1 ? '' : 's'))
    } else {
      state = state.set('activeItemSummary', 'nothing')
    }

    return state
  },

  chooseCommand(command) {
    const state = this.state.set('selectedCommand', command)
    this.triggerUpdate(this._updateFilter(state))
  },

  apply(commandParams) {
    const activeItems = this.state.items.filter(item => item.get('active'))
    commands[this.state.selectedCommand].execute(activeItems, commandParams)
  },
})
