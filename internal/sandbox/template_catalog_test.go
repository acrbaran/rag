package sandbox

import "testing"

func TestIsStandardTemplateRecognizesProviderScopedName(t *testing.T) {
	for _, name := range []string{"rethra", "team/rethra", "project-b89e/Rethra"} {
		if !isStandardTemplate(name) {
			t.Fatalf("expected %q to identify the Rethra standard template", name)
		}
	}
	if isStandardTemplate("rethra-custom") {
		t.Fatal("custom template must not be treated as the standard template")
	}
	if isStandardTemplate(DesktopTemplateName) {
		t.Fatal("the desktop sibling must not be classified as the CLI standard template")
	}
}

func TestIsDesktopTemplateRecognizesProviderScopedName(t *testing.T) {
	for _, name := range []string{"rethra-desktop", "team/rethra-desktop", "project-b89e/Rethra-Desktop"} {
		if !isDesktopTemplate(name) {
			t.Fatalf("expected %q to identify the Rethra desktop template", name)
		}
	}
	if isDesktopTemplate("rethra") {
		t.Fatal("the CLI template must not be classified as desktop")
	}
	if isDesktopTemplate("rethra-desktop-custom") {
		t.Fatal("a similarly prefixed custom name must not be the desktop template")
	}
}

func TestClassifyRethraTemplatePrefersNameOverImage(t *testing.T) {
	standard, desktop := classifyRethraTemplate(DesktopTemplateName, DefaultDockerImage)
	if standard || !desktop {
		t.Fatalf(
			"named desktop template must be desktop even if the image repo matches CLI, got standard=%v desktop=%v",
			standard, desktop,
		)
	}
	standard, desktop = classifyRethraTemplate(StandardTemplateName, DefaultDesktopDockerImage)
	if !standard || desktop {
		t.Fatalf(
			"named CLI template must stay CLI even if the image tag is desktop, got standard=%v desktop=%v",
			standard, desktop,
		)
	}
}

func TestClassifyRethraTemplateNamelessImageUsesTag(t *testing.T) {
	standard, desktop := classifyRethraTemplate("", DefaultDockerImage)
	if !standard || desktop {
		t.Fatalf("CLI image with no name must be standard, got standard=%v desktop=%v", standard, desktop)
	}
	standard, desktop = classifyRethraTemplate("", DefaultDesktopDockerImage)
	if standard || !desktop {
		t.Fatalf("desktop image with no name must be desktop, got standard=%v desktop=%v", standard, desktop)
	}
	standard, desktop = classifyRethraTemplate("", DefaultCubeDesktopTemplateImage)
	if standard || !desktop {
		t.Fatalf("Cube desktop image with no name must be desktop, got standard=%v desktop=%v", standard, desktop)
	}
}
