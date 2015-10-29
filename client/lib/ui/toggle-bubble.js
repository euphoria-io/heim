import _ from 'lodash'
import React from 'react/addons'

import Bubble from './bubble'


export default React.createClass({
  displayName: 'ToggleBubble',

  propTypes: {
    visible: React.PropTypes.bool,
    sticky: React.PropTypes.bool,
    children: React.PropTypes.node,
  },

  mixins: [require('react-immutable-render-mixin')],

  getInitialState() {
    return {visible: false}
  },

  componentWillMount() {
    // queue cancelable hide so that if the click triggers a show, we don't
    // hide and then immediately reshow.
    this._hide = _.debounce(this.hide, 0)
  },

  show() {
    this.setState({visible: true})
    this._hide.cancel()
  },

  hide() {
    this.setState({visible: false})
  },

  toggle() {
    if (this.state.visible) {
      this._hide()
    } else {
      this.show()
    }
  },

  render() {
    return (
      <Bubble {...this.props} visible={this.props.visible || this.state.visible} onDismiss={this.props.sticky ? null : this._hide}>
        {this.props.children}
      </Bubble>
    )
  },
})
