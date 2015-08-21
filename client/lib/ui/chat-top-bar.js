var React = require('react/addons')
var classNames = require('classnames')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup

var update = require('../stores/update')
var FastButton = require('./fast-button')
var Bubble = require('./toggle-bubble')
var RoomTitle = require('./room-title')


module.exports = React.createClass({
  displayName: 'ChatTopBar',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var userCount = this.props.who.filter(user => user.get('name')).size

    // use an outer container element so we can z-index the bar above the
    // bubbles. this makes the bubbles slide from "underneath" the bar.
    return (
      <div className="top-bar">
        {this.props.showInfoPaneButton && <FastButton className={classNames(this.props.infoPaneOpen ? 'collapse-info-pane' : 'expand-info-pane')} onClick={this.props.infoPaneOpen ? this.props.collapseInfoPane : this.props.expandInfoPane} />}
        <RoomTitle name={this.props.roomName} authType={this.props.authType} connected={this.props.connected} joined={this.props.joined} />
        <div className="right">
          <ReactCSSTransitionGroup transitionName="spinner">{this.props.working && <div key="spinner" className="spinner" />}</ReactCSSTransitionGroup>
          {this.props.joined && <FastButton fastTouch className="user-count" onClick={this.props.toggleUserList}>{userCount}</FastButton>}
        </div>
        <Bubble ref="updateBubble" className="update" visible={this.props.updateReady}>
          <FastButton className="update-button" onClick={update.perform}><p>update ready<em>{Heim.isTouch ? 'tap' : 'click'} to reload</em></p></FastButton>
        </Bubble>
      </div>
    )
  },
})
