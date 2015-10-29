import React from 'react'


export default React.createClass({
  propTypes: {
    cellPadding: React.PropTypes.number,
    cellSpacing: React.PropTypes.number,
    border: React.PropTypes.string,
    bgcolor: React.PropTypes.string,
    width: React.PropTypes.string,
    height: React.PropTypes.string,
    style: React.PropTypes.object,
    children: React.PropTypes.node,
    align: React.PropTypes.oneOf(['left', 'center', 'right']),
    valign: React.PropTypes.oneOf(['top', 'middle', 'bottom']),
  },

  getDefaultProps() {
    return {
      cellPadding: 0,
      cellSpacing: 0,
      border: '0',
      align: 'left',
      valign: 'top',
    }
  },

  render() {
    return (
      <table align={this.props.align} valign={this.props.valign} cellPadding={this.props.cellPadding} cellSpacing={this.props.cellSpacing} border={this.props.border} bgcolor={this.props.bgcolor} width={this.props.width} height={this.props.height} style={this.props.style}>
        {this.props.children}
      </table>
    )
  },
})
