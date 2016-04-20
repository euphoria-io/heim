import React from 'react'
import classNames from 'classnames'
import Reflux from 'reflux'

import chat from '../stores/chat'
import toolbox from '../stores/toolbox'
import FastButton from './FastButton'


export default React.createClass({
  displayName: 'ManagerToolbox',

  mixins: [
    Reflux.connect(chat.store, 'chat'),
    Reflux.connect(toolbox.store, 'toolbox'),
  ],

  selectCommand(ev) {
    toolbox.chooseCommand(ev.target.value)
  },

  apply() {
    let commandParams
    const selectedCommand = this.state.toolbox.selectedCommand
    if (selectedCommand === 'ban' || selectedCommand === 'banIP') {
      commandParams = {
        seconds: {
          h: 60 * 60,
          d: 24 * 60 * 60,
          w: 7 * 24 * 60 * 60,
          m: 30 * 24 * 60 * 60,
          f: null,
        }[this.refs.banDuration.value],
      }
    }
    if (selectedCommand === 'banIP') {
      commandParams = {
        global: this.refs.banGlobally && this.refs.banGlobally.checked,
      }
    }
    toolbox.apply(commandParams)
  },

  render() {
    const toolboxData = this.state.toolbox
    const isEmpty = !toolboxData.items.size
    const selectedCommand = this.state.toolbox.selectedCommand
    const inputDuration = selectedCommand === 'ban' || selectedCommand === 'banIP'
    return (
      <div className="manager-toolbox">
        <div className={classNames('items', {'empty': isEmpty})} onCopy={this.onCopy}>
          {isEmpty && 'nothing selected'}
          {toolboxData.items.toSeq().map(item =>
            <span key={item.get('kind') + '-' + item.get('id') + '-' + item.get('name', '')} className={classNames('item', item.get('kind'), {'active': item.get('active'), 'removed': item.get('removed')})}>
              {item.has('name') && <div className="name">{item.get('name')}</div>}
              <div className="id">{item.get('id')}</div>
              {item.has('addr') && <div className="addr">{item.get('realAddr') || item.get('addr')}</div>}
            </span>
          )}
        </div>
        <div className="action">
          <select className="command-picker" value={selectedCommand} onChange={this.selectCommand}>
            <option value="delete">delete</option>
            <option value="ban">ban</option>
            <option value="banIP">IP ban</option>
            <option value="pm">pm</option>
          </select>
          <div className="preview">{toolboxData.activeItemSummary}</div>
          {!isEmpty && inputDuration && <select ref="banDuration" defaultValue={60 * 60}>
            <option value="h">for 1 hour</option>
            <option value="d">for 1 day</option>
            <option value="w">for 1 week</option>
            <option value="m">for 30 days</option>
            <option value="f">forever</option>
          </select>}
          {!isEmpty && selectedCommand === 'banIP' && this.state.chat.isStaff && <label className="toggle-global"><input type="checkbox" ref="banGlobally" /> everywhere</label>}
          <div className="spacer" />
          <FastButton className="apply" onClick={this.apply}>
            <div className="emoji emoji-26a1" /> apply
          </FastButton>
        </div>
      </div>
    )
  },
})
