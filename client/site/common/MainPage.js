import React from 'react'
import classNames from 'classnames'

import Page from './Page'
import Header from './Header'
import Footer from './Footer'


export default React.createClass({
  propTypes: {
    className: React.PropTypes.string,
    title: React.PropTypes.string,
    nav: React.PropTypes.node,
    children: React.PropTypes.node,
  },

  render() {
    return (
      <Page className={classNames('page', this.props.className)} title={this.props.title}>
        <Header />
        {this.props.nav || null}
        <div className="container main">
          {this.props.children}
        </div>
        <Footer />
      </Page>
    )
  },
})

