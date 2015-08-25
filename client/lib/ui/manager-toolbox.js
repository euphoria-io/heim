var React = require('react/addons')
var classNames = require('classnames')
var Reflux = require('reflux')

var toolbox = require('../stores/toolbox')
var FastButton = require('./fast-button')
var domWalkForward = require('../dom-walk-forward')


module.exports = React.createClass({
  displayName: 'ManagerToolbox',

  mixins: [
    Reflux.connect(toolbox.store, 'toolbox'),
  ],

  selectCommand: function(ev) {
    toolbox.chooseCommand(ev.target.value)
  },

  apply: function() {
    var commandParams
    if (this.state.toolbox.selectedCommand == 'ban') {
      commandParams = {
        seconds: {
          h: 60 * 60,
          d: 24 * 60 * 60,
          w: 7 * 24 * 60 * 60,
          m: 30 * 24 * 60 * 60,
          f: null,
        }[this.refs.banDuration.getDOMNode().value],
      }
    }
    toolbox.apply(commandParams)
  },

  onCopy: function(ev) {
    var selection = uiwindow.getSelection()
    var range = selection.getRangeAt(0)

    var ids = []
    domWalkForward(range.startContainer, range.endContainer, function(childEl) {
      var el = childEl.parentNode
      if (!el.classList || !el.classList.contains('id')) {
        return
      }
      ids.push(el.textContent)
    })

    ev.clipboardData.setData('text/plain', "'" + ids.join("', '") + "'")
    ev.preventDefault()
  },

  render: function() {
    var toolboxData = this.state.toolbox
    var isEmpty = !toolboxData.items.size
    var selectedCommand = this.state.toolbox.selectedCommand
    return (
      <div className="manager-toolbox">
        <div className={classNames('items', {'empty': isEmpty})} onCopy={this.onCopy}>
          {isEmpty && 'nothing selected'}
          {toolboxData.items.toSeq().map(item =>
            <span key={item.get('kind') + '-' + item.get('id') + '-' + item.get('name', '')} className={classNames('item', item.get('kind'), {'active': item.get('active'), 'removed': item.get('removed')})}>
              {item.has('name') && <div className="name">{item.get('name')}</div>}
              <div className="id">{item.get('id')}</div>
            </span>
          ).toArray()}
        </div>
        <div className="action">
          <select className="command-picker" value={selectedCommand} onChange={this.selectCommand}>
            <option value="delete">delete</option>
            <option value="ban">ban</option>
          </select>
          <div className="preview">{toolboxData.activeItemSummary}</div>
          {!isEmpty && selectedCommand == 'ban' && <select ref="banDuration" defaultValue={60 * 60}>
            <option value="h">for 1 hour</option>
            <option value="d">for 1 day</option>
            <option value="w">for 1 week</option>
            <option value="m">for 30 days</option>
            <option value="f">forever</option>
          </select>}
          <div className="spacer" />
          <FastButton className="apply" onClick={this.apply}>
            <div className="emoji emoji-26a1" /> apply
          </FastButton>
        </div>
      </div>
    )
  },
})
