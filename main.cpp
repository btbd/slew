#include "main.h"

THREAD main_thread = { 0 };

int main(int argc, char **argv) {
	main_thread.stack = ArrayNew(sizeof(VALUE));
	SetupLibraries(&main_thread);

	if (argc < 2) {
		static char input[0xFFFF] = { 0 };

		for (char *buffer = input;; buffer = input) {
			fputs("> ", stdout);
			while ((*buffer++ = getchar()) != '\n');
			*buffer = 0;

			ARRAY tokens = GetTokens(input);
			if (tokens.length > 0) {
				for (unsigned int i = 0; i < tokens.length; ++i) {
					TREE *tree = CreateTree(&tokens, &i, false);
					if (tree) {
						// PrintTree(tree);
						VALUE value = Eval(&main_thread, tree, 0);
						fputs("< ", stdout);
						PrintValue(&value, 1);
						putchar('\n');
						ValueFree(&value);
						FreeTree(tree);
					}
				}

				for (unsigned int t = 0; t < tokens.length; ++t) {
					free(((TOKEN *)ArrayGet(&tokens, t))->value);
				}
			}
			ArrayFree(&tokens);
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

		ARRAY tokens = GetTokens(buffer);
		if (tokens.length > 0) {
			for (unsigned int i = 0; i < tokens.length; ++i) {
				TREE *tree = CreateTree(&tokens, &i, false);
				if (tree) {
					VALUE value = Eval(&main_thread, tree, 0);
					ValueFree(&value);
					FreeTree(tree);
				}
			}

			for (unsigned int t = 0; t < tokens.length; ++t) {
				free(((TOKEN *)ArrayGet(&tokens, t))->value);
			}
		}
		ArrayFree(&tokens);

		free(buffer);
	}

	return 0;
}

THREAD *GetMainThread() {
	return &main_thread;
}