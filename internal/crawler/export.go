package crawler

import (
	"encoding/csv"
	"os"
	"strings"
)

func NormalizeProfileURL(raw string) string {
	if raw == "" {
		return raw
	}
	raw = strings.Split(raw, "?")[0]
	i := strings.Index(raw, "/in/")
	if i == -1 {
		return raw
	}
	return raw[:i+4] + strings.Trim(raw[i+4:], "/")
}

func RemoveDup(cs []Contact) []Contact {
	seen := map[string]bool{}
	out := make([]Contact, 0, len(cs))
	for _, c := range cs {
		k := strings.ToLower(c.LinkedIn)
		if k == "" {
			k = strings.ToLower(c.Name + "|" + c.Company)
		}
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, c)
		}
	}
	return out
}

func SaveCSV(path string, cs []Contact) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"Nome", "Cargo", "Empresa", "Localização", "LinkedIn"})
	for _, c := range cs {
		_ = w.Write([]string{c.Name, c.Title, c.Company, c.Location, c.LinkedIn})
	}
	return nil
}



