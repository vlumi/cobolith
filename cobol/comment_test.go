package cobol

import (
	"slices"
	"testing"
)

func TestCommentText(t *testing.T) {
	banner := []string{
		"================================================================*",
		" TXSORT: SORT THE VALID TRANSACTIONS FOR POSTING                *",
		"                                                                *",
		"                                                                *",
		" RETURN CODES: 0 SORTED, 16 SORT FAILED.                        *",
		"                                                                *",
		"================================================================*",
	}
	want := []string{"TXSORT: SORT THE VALID TRANSACTIONS FOR POSTING", "", "RETURN CODES: 0 SORTED, 16 SORT FAILED."}
	if got := CommentText(banner); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := CommentText([]string{"    READ THE NEXT RECORD."}); !slices.Equal(got, []string{"READ THE NEXT RECORD."}) {
		t.Errorf("plain comment %q", got)
	}
}
