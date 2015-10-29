import _ from 'lodash'
import Immutable from 'immutable'
import EventEmitter from 'eventemitter3'


export default class Tree {
  constructor(sortProp, updateFunc) {
    this.sortProp = sortProp
    if (!this.updateFunc) {
      // subclasses can define updateFunc in their prototype
      this.updateFunc = updateFunc || _.noop
    }
    this.changes = new EventEmitter()
    this.reset()
  }

  _node(entry = {}) {
    return Immutable.fromJS(entry)
      .merge({
        'children': Immutable.OrderedSet(),
      })
  }

  _add(entry, _changed, _needsSort) {
    if (entry.parent === undefined) {
      entry.parent = '__root'
    }

    const newId = entry.id
    let newNode = this._node(entry)
    const parentId = entry.parent

    if (parentId) {
      const parentNode = this.index[parentId]
      if (parentNode) {
        const siblings = parentNode.get('children')
        const canAppend = siblings.size === 0 || entry[this.sortProp] >= this.index[siblings.last()].get(this.sortProp)
        if (canAppend) {
          // avoiding a re-sort can save a lot of time with large child lists
          this.index[parentId] = parentNode.set('children', siblings.delete(newId).add(newId))
          if (!Immutable.is(parentNode, this.index[parentId])) {
            _changed[parentId] = _changed[parentId] || parentNode
          }
        } else {
          this.index[parentId] = parentNode.set('children', siblings.add(newId))
          _needsSort[parentId] = true
          _changed[parentId] = _changed[parentId] || parentNode
        }
      } else {
        // create unreachable orphan parent
        this.index[parentId] = this._node().set('id', parentId).mergeIn(['children'], [newId])
        _changed[parentId] = _changed[parentId] || true
        this.size++
        _needsSort[parentId] = true
      }
    }

    if (_.has(this.index, newId)) {
      const oldNode = this.index[newId]

      const oldParentId = oldNode.get('parent')
      const oldParent = oldParentId && oldParentId !== parentId && this.index[oldParentId]
      if (oldParent) {
        this.index[oldParentId] = oldParent.set('children', oldParent.get('children').remove(newId))
        _changed[oldParentId] = _changed[oldParentId] || oldParent
      }

      // merge in orphans
      newNode = oldNode.mergeDeep(newNode)
      if (!_changed[newId] && !Immutable.is(oldNode, newNode)) {
        _changed[newId] = oldNode
      }
    } else {
      _changed[newId] = _changed[newId] || true
      this.size++
    }

    if (_changed[newId]) {
      this.index[newId] = newNode

      if (entry[this.sortProp] > this._lastValue) {
        this._lastId = newId
        this._lastValue = entry[this.sortProp]
      }
    }
  }

  _updateChanged(changed) {
    if (!_.isEmpty(changed)) {
      const updateNode = node => {
        const nodeId = node.get('id')
        const existing = this.index[nodeId]
        if (!Immutable.is(node, existing)) {
          changed[nodeId] = changed[nodeId] || existing
          this.index[nodeId] = node
        }
        return node
      }
      this.updateFunc(changed, updateNode)

      const changedIds = _.keys(changed)
      _.each(changedIds, id => {
        this.changes.emit(id, this.index[id])
      })
      this.changes.emit('__all', changedIds)
    }
  }

  add(entries, _sorted) {
    const entryArray = _.isArray(entries) ? entries : [entries]

    const _changed = {}
    const _needsSort = {}
    _.each(entryArray, entry => {
      this._add(entry, _changed, _needsSort)
    })

    if (!_sorted) {
      _.each(_needsSort, (x, id) => {
        const resorted = this.index[id].get('children').sortBy(childId => {
          return this.index[childId].get(this.sortProp)
        })
        this.index[id] = this.index[id].set('children', resorted)

        // if parent now matches the original one, no need to emit the change
        if (Immutable.is(_changed[id], this.index[id])) {
          delete _changed[id]
        }
      })
    }

    this._updateChanged(_changed)
  }

  mergeNodes(ids, data) {
    const idArray = _.isArray(ids) ? ids : [ids]

    const changed = {}
    _.each(idArray, id => {
      const old = this.index[id]
      this.index[id] = this.index[id].mergeDeep(data)
      if (old !== this.index[id]) {
        changed[id] = old
      }
    })
    this._updateChanged(changed)
  }

  reset(entries) {
    this.index = {}
    this.index.__root = this._node({id: '__root'})
    this._lastId = null
    this._lastValue = null
    this.size = 0

    if (entries) {
      this.add(entries, true)
    } else {
      this.changes.emit('__root', this.index.__root)
      this.changes.emit('__all', ['__root'])
    }
    return this
  }

  get(id) {
    return this.index[id]
  }

  lazyMapDFS(visit, thisArg, nodeId = '__root', depth = 0) {
    const node = this.index[nodeId]
    const children = node.get('children').toSeq().map((childId) =>
      this.lazyMapDFS(visit, thisArg, childId, depth + 1)
    )

    return visit.call(thisArg, node, children, depth)
  }

  mapDFS(visit, thisArg, nodeId, depth) {
    const visitStrict = (node, children, vdepth) => visit.call(thisArg, node, children.cacheResult(), vdepth)
    return this.lazyMapDFS(visitStrict, thisArg, nodeId, depth)
  }

  iterChildrenOf(nodeId) {
    const children = this.index[nodeId].get('children').toJS()
    return {
      next: () => {
        const childId = children.shift()
        if (!childId) {
          return {done: true}
        }
        return {done: false, value: this.index[childId]}
      },
    }
  }

  iterAncestorsOf(nodeId) {
    let curNodeId = nodeId
    return {
      next: () => {
        curNodeId = this.index[curNodeId].get('parent')
        if (!curNodeId) {
          return {done: true}
        }
        return {done: false, value: this.index[curNodeId]}
      },
    }
  }

  last() {
    return this.index[this._lastId]
  }

  clone() {
    const newTree = new Tree(this.sortProp, this.updateFunc)
    newTree.index = _.clone(this.index)
    newTree._lastId = this._lastId
    newTree._lastValue = this._lastValue
    newTree.size = this.size
    return newTree
  }
}
