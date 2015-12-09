import plugins from '../stores/plugins'


export default {
  templateHook(name) {
    return plugins.hooks.run(name, this)
  },
}
