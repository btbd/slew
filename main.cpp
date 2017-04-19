#include "main.h"

int main(int argc, char *argv[]) {
	SetupLibraries();

	if (argc < 2) {
		char input[0xFFFF];

		for (char *buffer = input;; buffer = input) {
			fputs("> ", stdout);
			while ((*buffer++ = getchar()) != '\n');
			*buffer = 0;

			ARRAY *tokens = GetTokens(input);
			if (tokens) {
				unsigned int i = 0;
				for (; i < tokens->length; ++i) {
					TREE *tree = CreateTree(tokens, &i);
					if (tree) {
						PrintTree(tree);
						VALUE value = Eval(tree, 0);
						fputs("< ", stdout);
						PrintValue(&value);
						putchar('\n');
						Value_Free(&value);
						free(tree);
					}
				}
				for (i = 0; i < tokens->length; ++i) {
					free(((TOKEN *)Array_Get(tokens, i))->value);
				}
				Array_Free(tokens);
			}
		}
	} else {
		FILE *file = fopen(argv[1], "rb");
		if (!file) {
			printf("error: unable to open '%s'", argv[1]);
			return 1;
		}
		fseek(file, 0, SEEK_END);
		unsigned int length = ftell(file);
		fseek(file, 0, SEEK_SET);

		char *buffer = (char *)calloc(length + 1, 1);
		fread(buffer, length, 1, file);
		fclose(file);

		ARRAY *tokens = GetTokens(buffer);
		if (tokens) {
			unsigned int i = 0;
			for (; i < tokens->length; ++i) {
				TREE *tree = CreateTree(tokens, &i);

				if (tree) {
					VALUE value = Eval(tree, 0);
					Value_Free(&value);
					free(tree);
				}
			}
			for (i = 0; i < tokens->length; ++i) {
				free(((TOKEN *)Array_Get(tokens, i))->value);
			}
			Array_Free(tokens);
		}
	}

	return 0;
}