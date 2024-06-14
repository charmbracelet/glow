package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/charm"
)

func Test_markdownsByLocalFirst_Less(t *testing.T) {
	a := charm.Markdown{
		ID:           0,
		EncryptKeyID: "blabla",
		Note:         "Blabla",
		Body:         "blabla",
		CreatedAt:    time.Now().Local().Add(time.Hour * time.Duration(10)),
	}

	b := charm.Markdown{
		ID:           1,
		EncryptKeyID: "blabla2",
		Note:         "Blabla2",
		Body:         "blabla2",
		CreatedAt:    time.Now().Local(),
	}

	t.Run("Should returns true if the first markdown is local and the second is not local", func(f *testing.T) {
		var markdowns markdownsByLocalFirst

		markdowns = append(markdowns, &markdown{
			localMarkdown,
			"blabla",
			"blabla",
			a,
		})

		markdowns = append(markdowns, &markdown{
			stashedMarkdown,
			"blabla2",
			"blabla2",
			b,
		})

		if ok := markdowns.Less(0, 1); !ok {

			t.Errorf("We got %t, expecting true", ok)
		}
	})

	t.Run("Should returns false if the first markdown is not local and the second is local", func(f *testing.T) {
		var markdowns markdownsByLocalFirst

		markdowns = append(markdowns, &markdown{
			stashedMarkdown,
			"blabla",
			"blabla",
			a,
		})

		markdowns = append(markdowns, &markdown{
			localMarkdown,
			"blabla2",
			"blabla2",
			b,
		})

		if ok := markdowns.Less(0, 1); ok {

			t.Errorf("We got %t, expecting false", ok)
		}
	})

	t.Run("Should returns true if both markdowns are local and the name of the first markdown comes first alphabetically compared with the second one", func(f *testing.T) {
		var markdowns markdownsByLocalFirst

		markdowns = append(markdowns, &markdown{
			localMarkdown,
			"blabla",
			"blabla",
			a,
		})

		markdowns = append(markdowns, &markdown{
			localMarkdown,
			"blabla2",
			"blabla2",
			b,
		})

		if ok := markdowns.Less(0, 1); !ok {

			t.Errorf("We got %t, expecting true", ok)
		}
	})

	t.Run("Should returns true if both markdowns are not local and the first markdown has been made after the second one", func(f *testing.T) {
		var markdowns markdownsByLocalFirst

		markdowns = append(markdowns, &markdown{
			stashedMarkdown,
			"blabla",
			"blabla",
			a,
		})

		markdowns = append(markdowns, &markdown{
			stashedMarkdown,
			"blabla2",
			"blabla2",
			b,
		})

		if ok := markdowns.Less(0, 1); !ok {

			t.Errorf("We got %t, expecting true", ok)
		}
	})

	t.Run("Should returns true if both markdowns are not local and made at the same time and the first markdown has a higher ID than the second one", func(f *testing.T) {
		createdAt := time.Now().Local()

		a := charm.Markdown{
			ID:           2,
			EncryptKeyID: "blabla",
			Note:         "Blabla",
			Body:         "blabla",
			CreatedAt:    createdAt,
		}

		b := charm.Markdown{
			ID:           1,
			EncryptKeyID: "blabla2",
			Note:         "Blabla2",
			Body:         "blabla2",
			CreatedAt:    createdAt,
		}

		var markdowns markdownsByLocalFirst

		markdowns = append(markdowns, &markdown{
			stashedMarkdown,
			"blabla",
			"blabla",
			a,
		})

		markdowns = append(markdowns, &markdown{
			stashedMarkdown,
			"blabla2",
			"blabla2",
			b,
		})

		if ok := markdowns.Less(0, 1); !ok {

			t.Errorf("We got %t, expecting true", ok)
		}
	})
}
