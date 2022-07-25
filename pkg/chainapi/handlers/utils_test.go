package handlers

import (
	"testing"
)

func TestUrlToGroupSeed(t *testing.T) {
	//url := "rum://seed?v=1&e=0&b=VcxDHxv5SgKTraKbmVt9Qw&c=puq1Mlvkv4Hy-J6NwzqUZHnXKTLr-P16i7VH_9Bn_mw&g=IxYjCaT6SLCP3z2YVJRYBg&k=CAISIQPkmgFL5D4btXcF7R8UD101i_186HAl2WWL8RSqTWZmxA&s=MEYCIQDzXvq3kzir7lw0MhgE8iAxNzV6SXgoCgyQJ09mi5UDgQIhAPgG0LMT-EjfNtoGHqAvqfHChHrXw9q6J2EG1suA0PtR&t=Fvl8Us8CsHI&a=my_test_group%E4%B8%AD%E6%96%87&u=127.0.0.1%3A8080"

	url := "rum://seed?v=1&e=0&n=0&b=AEEmbwS1TBmjWVb9DW66KA&c=J1qik-4-ZElyhSGLnV4Gl3MRjGOnOIiLtrzEe7WqpoM&g=Wl-JkJR3Qfut52MqeNpmLg&k=CAISIQIiN0qnKdEhpDvfFdwdap9aSXoUZh99mUE0ED789AzahA&s=MEQCIBo4nZgGQiUlu5hu_VsFVkJUnkOlp0ZdQuXwcPeAPNV7AiAUjeuOjhTaGZbyKojAOQx3Ps_sVygIxhoHuKulFxADfw&t=FvppFL7rjgg&a=my_test_group%E4%B8%AD%E6%96%871&y=test_app&u=127.0.0.1%3A8080"
	seed, err := UrlToGroupSeed(url)
	if err != nil {
		t.Errorf("UrlToGroupSeed failed: %s", err)
	}

	_ = seed
	//verify
}
