import React from 'react'

import { Email } from '../email'


export default React.createClass({
  propTypes: {
    children: React.PropTypes.node,
  },

  render() {
    return (
      <Email title="{{.Subject}}" bgcolor="#f0f0f0" cellSpacing={30}>
        {this.props.children}
      </Email>
    )
  },
})
