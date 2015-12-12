import Immutable from 'immutable'

import { postAPI } from '../fetchAPI'


export default {
  _postAPI(url, data) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))

    return postAPI(url, data)
      .then(response => {
        if (response.error) {
          this.triggerUpdate(this.state.merge({
            working: false,
            errors: Immutable.Map({reason: response.error}),
          }))
        } else {
          this.triggerUpdate(this.state.merge({
            working: false,
            done: true,
          }))
        }
      })
      .catch(err => {
        Raven.captureException(err)
      })
  },
}
