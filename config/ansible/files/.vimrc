" Basic
if has('syntax')
	syntax on
endif
set number       " turn on line numbers by default
set backspace=2  " make backspace work in Insert mode (bs=2)
set nocompatible " don't try to be compatible with vi
filetype indent on

" Tabs
set tabstop=4
set shiftwidth=4
set softtabstop=4
set expandtab
set smartindent
set smarttab

" Searching
set path=,.** " recursive file search
set hlsearch  " search highlighting
set incsearch " get partial results while searching

" Tab completion
set wildmenu      " get suggesions on <tab>
set wildmode=longest:full,full

" Windows
set splitright
set splitbelow

" Misc
set mouse=         " disable mouse clicks and scrolling
set nofixendofline " stop vim from adding a newline at the end

" Key Maps
let mapleader=","
" Remap CTRL-j to Esc
inoremap <C-j> <Esc>
vnoremap <C-j> <Esc>
cnoremap <C-j> <C-C>
nnoremap <C-j> <Esc>

" Colors
"colorscheme delek " set delek as a fallback