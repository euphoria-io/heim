import React from 'react'
import classNames from 'classnames'

import Page from './Page'
import Footer from './Footer'


export default function MainPage(props) {
  return (
    <Page className={classNames('page', props.className)} title={props.title} heimPage={props.heimPage}>
      {props.nav || null}
      <div className="container main">
        {props.children}
      </div>
      {props.bottom && props.bottom}
      <Footer noDonation={props.noDonation}/>
    </Page>
  )
}

MainPage.propTypes = {
  className: React.PropTypes.string,
  title: React.PropTypes.string,
  heimPage: React.PropTypes.string,
  nav: React.PropTypes.node,
  children: React.PropTypes.node,
  bottom: React.PropTypes.node,
  noDonation: React.PropTypes.bool,
}
