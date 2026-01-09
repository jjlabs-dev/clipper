#ifndef PASTEBOARD_H
#define PASTEBOARD_H

#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void* data;
    int length;
    char* uti;
} PasteboardContent;

PasteboardContent getPasteboardContent(void);
void setPasteboardData(const void* data, int length, const char* uti);
void emitCopy(void);
void emitPaste(void);
void freeContent(PasteboardContent c);

#endif
