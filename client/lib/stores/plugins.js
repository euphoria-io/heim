import _ from 'lodash'
import Reflux from 'reflux'

import Hooks from '../hooks'
import fauxPlugins from '../faux-plugins'


const storeActions = Reflux.createActions([
  'load',
])
_.extend(module.exports, storeActions)

const hooks = module.exports.hooks = new Hooks(
  'page-bottom',
  'main-sidebar',
  'thread-panes',
  'incoming-messages',
  'main-pane-top'
)

module.exports.hook = hooks.register.bind(hooks)

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  load(roomName) {
    fauxPlugins(roomName)
  },
})
