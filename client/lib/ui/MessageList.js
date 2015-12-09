import React from 'react'

import Message from './message'
import Tree from '../tree'
import { Pane } from '../stores/ui'
import TreeNodeMixin from './TreeNodeMixin'


export default React.createClass({
  displayName: 'MessageList',

  propTypes: {
    pane: React.PropTypes.instanceOf(Pane).isRequired,
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    showTimeStamps: React.PropTypes.bool,
    roomSettings: React.PropTypes.object,
  },

  mixins: [
    require('react-immutable-render-mixin'),
    TreeNodeMixin(),
  ],

  getDefaultProps() {
    return {nodeId: '__root', depth: 0}
  },

  componentDidMount() {
    this.props.pane.messageRenderFinished()
  },

  componentDidUpdate() {
    this.props.pane.messageRenderFinished()
  },

  render() {
    const children = this.state.node.get('children')
    return (
      <div className="message-list">
        {children.toIndexedSeq().map((nodeId, idx) =>
          <Message key={nodeId} pane={this.props.pane} tree={this.props.tree} nodeId={nodeId} showTimeAgo={idx === children.size - 1} showTimeStamps={this.props.showTimeStamps} roomSettings={this.props.roomSettings} />
        )}
      </div>
    )
  },
})
