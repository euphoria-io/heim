var _ = require('lodash')
var Immutable = require('immutable')
var EventEmitter = require('eventemitter3')


function Tree(entries) {
  this.changes = new EventEmitter()
  this.reset(entries || [])
}

_.extend(Tree.prototype, {
  _node: function(entry) {
    return Immutable.fromJS(entry)
      .merge({
        'children': Immutable.OrderedSet(),
      })
  },

  add: function(entry, silent) {
    if (!entry.parent) {
      entry.parent = '__root'
    }

    var newNode = this._node(entry)
    var newId = newNode.get('id')

    var parentId = newNode.get('parent')
    var parentNode = this.index[parentId]
    if (!parentNode) {
      // FIXME: just throwing away nodes with missing parents for now.
      return
    }
    parentNode = this.index[parentId] = parentNode.mergeIn(['children'], [newId])
    this.index[newId] = newNode
    this._lastId = newId
    this.size++

    if (!silent) {
      this.changes.emit(parentId, parentNode)
    }
  },

  mergeNode: function(id, data) {
    this.index[id] = this.index[id].merge(data)
    this.changes.emit(id, this.index[id])
  },

  reset: function(entries) {
    this.index = {}
    this.index.__root = this._node({id: '__root'})
    this._lastId = null
    this.size = 0

    _.each(entries, function(entry) {
      this.add(entry, true)
    }, this)
    this.changes.emit('__root', this.index.__root)
  },

  get: function(id) {
    return this.index[id]
  },

  mapDFS: function(visit, thisArg, nodeId, depth) {
    if (!nodeId) {
      nodeId = '__root'
    }

    if (depth === undefined) {
      depth = 0
    }

    var node = this.index[nodeId]
    var children = node.get('children').map(function(childId) {
      return this.mapDFS(visit, thisArg, childId, depth + 1)
    }, this)

    return visit.call(thisArg, node, children, depth)
  },

  last: function() {
    return this.index[this._lastId]
  }
})

module.exports = Tree
