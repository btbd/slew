#pragma once

typedef struct TREE TREE;

struct TREE {
	TOKEN *token;
	TREE *left, *right;
};

TREE *CreateTree(ARRAY *tokens, unsigned int *index, bool sign);
TREE *CopyTree(TREE *tree);
void FreeTree(TREE *tree);
void PrintTree(TREE *tree);
void _PrintTree(TREE *tree, wchar_t *prefix, int tail);
