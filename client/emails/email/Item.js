import React from 'react'


export default React.createClass({
  propTypes: {
    bgcolor: React.PropTypes.string,
    style: React.PropTypes.object,
    children: React.PropTypes.node,
    align: React.PropTypes.oneOf(['left', 'center', 'right']),
    valign: React.PropTypes.oneOf(['top', 'middle', 'bottom']),
  },

  render() {
    return (
      <tr>
        <td align={this.props.align} valign={this.props.valign} bgcolor={this.props.bgcolor} style={this.props.style}>
          {this.props.children}
        </td>
      </tr>
    )
  },
})
