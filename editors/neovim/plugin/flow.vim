" flow.nvim - AI coding assistant for Neovim
" Maintainer: flow-cli
" License: MIT

if exists('g:loaded_flow')
    finish
endif
let g:loaded_flow = 1

" Ensure Lua is available
if !has('nvim-0.8')
    echohl ErrorMsg
    echom "flow.nvim requires Neovim 0.8+"
    echohl None
    finish
endif

" Plugin will be initialized via lua require('flow').setup()
