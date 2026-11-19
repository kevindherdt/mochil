package appconfig

import "testing"

func TestAndroidModuleRunsUnlessSwitchedOff(t *testing.T) {
	if !(Config{}).ModuleEnabled("android") {
		t.Fatal("existing installs keep the Android app working")
	}
	if (Config{Modules: map[string]bool{"android": false}}).ModuleEnabled("android") {
		t.Fatal("an explicit off must win")
	}
	if (Config{}).ModuleEnabled("video") {
		t.Fatal("video still follows its libraries")
	}
}

// The two Hardcover extras are switches on the Hardcover card, not peer cards.
func TestHardcoverExtrasGroupUnderOneCard(t *testing.T) {
	byID := map[string]Module{}
	for _, module := range Modules {
		byID[module.ID] = module
	}
	lead, ok := byID["hardcover"]
	if !ok {
		t.Fatal("the hardcover module disappeared")
	}
	if lead.Group != "" || lead.Option {
		t.Fatalf("the hardcover module itself is the card, not an option on one: %+v", lead)
	}
	for _, id := range []string{"hardcover_progress", "hardcover_ratings"} {
		option, ok := byID[id]
		if !ok {
			t.Fatalf("%s disappeared", id)
		}
		if option.Group != "hardcover" || !option.Option {
			t.Fatalf("%s = group %q option %v, want group hardcover option true", id, option.Group, option.Option)
		}
		if option.Name == "" || option.Description == "" {
			t.Fatalf("%s still needs a name and description: %+v", id, option)
		}
	}
}

func TestGroupedHardcoverOptionsKeepTheirOwnSwitch(t *testing.T) {
	if !(Config{Modules: map[string]bool{"hardcover_progress": true}}).ModuleEnabled("hardcover_progress") {
		t.Fatal("modules.hardcover_progress:true must still switch the option on")
	}
	if !(Config{Modules: map[string]bool{"hardcover_ratings": true}}).ModuleEnabled("hardcover_ratings") {
		t.Fatal("modules.hardcover_ratings:true must still switch the option on")
	}
	if (Config{}).ModuleEnabled("hardcover_progress") {
		t.Fatal("grouping must not switch an option on by itself")
	}
	if (Config{Modules: map[string]bool{"hardcover": false, "hardcover_progress": true}}).ModuleEnabled("hardcover") {
		t.Fatal("switching the card off must stay off")
	}
	if err := (Config{Modules: map[string]bool{"hardcover_progress": true}}).validateModules(); err != nil {
		t.Fatalf("the grouped option is still a known module: %v", err)
	}
}
