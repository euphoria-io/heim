import React from 'react'


export default React.createClass({
  propTypes: {
    src: React.PropTypes.string.isRequired,
    width: React.PropTypes.number.isRequired,
    height: React.PropTypes.number.isRequired,
    style: React.PropTypes.object,
    children: React.PropTypes.node,
  },

  render() {
    return (
      <img src={this.props.src} width={this.props.width} height={this.props.height} style={{
        display: 'block',
        outline: 'none',
        border: 'none',
        textDecoration: 'none',
        ...this.props.style,
      }}>{this.props.children}</img>
    )
  },
})
