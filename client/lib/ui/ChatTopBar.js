const fs = require('fs')  // needs to be a require to work with brfs for now: https://github.com/babel/babelify/issues/81
import React from 'react'
import classNames from 'classnames'
import Immutable from 'immutable'

import FastButton from './FastButton'
import RoomTitle from './RoomTitle'
import Spinner from './Spinner'


const hexLeftSVG = fs.readFileSync(__dirname + '/../../res/hex-left-side.svg')
const hexRightSVG = hexLeftSVG.toString().replace('transform=""', 'transform="translate(7, 0) scale(-1, 1)"')

export default React.createClass({
  displayName: 'ChatTopBar',

  propTypes: {
    who: React.PropTypes.instanceOf(Immutable.Map),
    showInfoPaneButton: React.PropTypes.bool,
    infoPaneOpen: React.PropTypes.bool,
    collapseInfoPane: React.PropTypes.func,
    expandInfoPane: React.PropTypes.func,
    toggleUserList: React.PropTypes.func,
    roomName: React.PropTypes.string.isRequired,
    roomTitle: React.PropTypes.string.isRequired,
    authType: React.PropTypes.string,
    connected: React.PropTypes.bool,
    joined: React.PropTypes.bool,
    isManager: React.PropTypes.bool,
    managerMode: React.PropTypes.bool,
    toggleManagerMode: React.PropTypes.func,
    working: React.PropTypes.bool,
  },

  mixins: [require('react-immutable-render-mixin')],

  render() {
    let people = this.props.who.filter(user =>
      user.get('present') && user.get('name') && !/^bot:/.test(user.get('id')))
    let prevUser
    people = people.filter(user => {
      if (prevUser && user.get('id') === prevUser.get('id') && user.get('name') === prevUser.get('name')) {
        return false
      }
      prevUser = user
      return true
    })
    const userCount = people.size

    // use an outer container element so we can z-index the bar above the
    // bubbles. this makes the bubbles slide from "underneath" the bar.
    return (
      <div className="top-bar">
        {this.props.showInfoPaneButton && <FastButton className={classNames(this.props.infoPaneOpen ? 'collapse-info-pane' : 'expand-info-pane')} onClick={this.props.infoPaneOpen ? this.props.collapseInfoPane : this.props.expandInfoPane} />}
        <RoomTitle name={this.props.roomName} title={this.props.roomTitle} authType={this.props.authType} connected={this.props.connected} joined={this.props.joined} />
        {this.props.isManager && <FastButton className={classNames('manager-toggle', {'on': this.props.managerMode})} onClick={this.props.toggleManagerMode}><div className="hex left" dangerouslySetInnerHTML={{__html: hexLeftSVG}} />{this.props.managerMode ? 'host mode' : 'host'}<div className="hex right" dangerouslySetInnerHTML={{__html: hexRightSVG}} /></FastButton>}
        <div className="right">
          <Spinner visible={this.props.working} />
          {this.props.joined && <FastButton fastTouch className="user-count" onClick={this.props.toggleUserList}>{userCount}</FastButton>}
        </div>
      </div>
    )
  },
})
