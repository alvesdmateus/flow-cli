" vibe.nvim - AI coding assistant for Neovim
" Maintainer: vibe-cli
" License: MIT

if exists('g:loaded_vibe')
    finish
endif
let g:loaded_vibe = 1

" Ensure Lua is available
if !has('nvim-0.8')
    echohl ErrorMsg
    echom "vibe.nvim requires Neovim 0.8+"
    echohl None
    finish
endif

" Plugin will be initialized via lua require('vibe').setup()
