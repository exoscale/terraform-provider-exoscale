package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	testIPHealthcheckMode1              = "http"
	testIPHealthcheckPort1        int64 = 80
	testIPHealthcheckPath1              = "/health"
	testIPHealthcheckInterval1    int64 = 10
	testIPHealthcheckTimeout1     int64 = 5
	testIPHealthcheckStrikesOk1   int64 = 1
	testIPHealthcheckStrikesFail1       = 2
	testIPHealthcheckMode2              = "http"
	testIPHealthcheckPort2        int64 = 8000
	testIPHealthcheckPath2              = "/healthz"
	testIPHealthcheckInterval2    int64 = 5
	testIPHealthcheckTimeout2     int64 = 2
	testIPHealthcheckStrikesOk2   int64 = 2
	testIPHealthcheckStrikesFail2 int64 = 3
)

func TestAccResourceIPAddress(t *testing.T) {
	eip := new(egoscale.IPAddress)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIPAddressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIPAddressConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPAddressExists("exoscale_ipaddress.eip", eip),
					testAccCheckIPAddressCreate(eip),
					testAccCheckIPAddressAttributes(testAttrs{
						"healthcheck_mode":         ValidateString(testIPHealthcheckMode1),
						"healthcheck_port":         ValidateString(fmt.Sprint(testIPHealthcheckPort1)),
						"healthcheck_path":         ValidateString(testIPHealthcheckPath1),
						"healthcheck_interval":     ValidateString(fmt.Sprint(testIPHealthcheckInterval1)),
						"healthcheck_timeout":      ValidateString(fmt.Sprint(testIPHealthcheckTimeout1)),
						"healthcheck_strikes_ok":   ValidateString(fmt.Sprint(testIPHealthcheckStrikesOk1)),
						"healthcheck_strikes_fail": ValidateString(fmt.Sprint(testIPHealthcheckStrikesFail1)),
						"description":              ValidateString("IPAdress 1"),
					}),
				),
			},
			{
				Config: testAccIPAddressConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPAddressExists("exoscale_ipaddress.eip", eip),
					testAccCheckIPAddressUpdate(eip),
					testAccCheckIPAddressAttributes(testAttrs{
						"healthcheck_mode":         ValidateString(testIPHealthcheckMode2),
						"healthcheck_port":         ValidateString(fmt.Sprint(testIPHealthcheckPort2)),
						"healthcheck_path":         ValidateString(testIPHealthcheckPath2),
						"healthcheck_interval":     ValidateString(fmt.Sprint(testIPHealthcheckInterval2)),
						"healthcheck_timeout":      ValidateString(fmt.Sprint(testIPHealthcheckTimeout2)),
						"healthcheck_strikes_ok":   ValidateString(fmt.Sprint(testIPHealthcheckStrikesOk2)),
						"healthcheck_strikes_fail": ValidateString(fmt.Sprint(testIPHealthcheckStrikesFail2)),
						"description":              ValidateString("IPAdress 1 updated!"),
					}),
				),
			},
			{
				ResourceName:      "exoscale_ipaddress.eip",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"healthcheck_mode":         ValidateString(testIPHealthcheckMode2),
							"healthcheck_port":         ValidateString(fmt.Sprint(testIPHealthcheckPort2)),
							"healthcheck_path":         ValidateString(testIPHealthcheckPath2),
							"healthcheck_interval":     ValidateString(fmt.Sprint(testIPHealthcheckInterval2)),
							"healthcheck_timeout":      ValidateString(fmt.Sprint(testIPHealthcheckTimeout2)),
							"healthcheck_strikes_ok":   ValidateString(fmt.Sprint(testIPHealthcheckStrikesOk2)),
							"healthcheck_strikes_fail": ValidateString(fmt.Sprint(testIPHealthcheckStrikesFail2)),
							"description":              ValidateString("IPAdress 1 updated!"),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckIPAddressExists(n string, eip *egoscale.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := GetComputeClient(testAccProvider.Meta())
		eip.ID = id
		resp, err := client.Get(eip)
		if err != nil {
			return err
		}

		return Copy(eip, resp.(*egoscale.IPAddress))
	}
}

func testAccCheckIPAddressCreate(eip *egoscale.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if eip.IPAddress == nil {
			return errors.New("IP address is nil")
		}

		if eip.Healthcheck == nil {
			return errors.New("IP healthcheck is nil")
		}
		if eip.Healthcheck.Mode != testIPHealthcheckMode1 {
			return fmt.Errorf("expected IP healthcheck mode %v, got %v",
				testIPHealthcheckMode1,
				eip.Healthcheck.Mode)
		}
		if eip.Healthcheck.Port != testIPHealthcheckPort1 {
			return fmt.Errorf("expected IP healthcheck port %v, got %v",
				testIPHealthcheckPort1,
				eip.Healthcheck.Port)
		}
		if eip.Healthcheck.Path != testIPHealthcheckPath1 {
			return fmt.Errorf("expected IP healthcheck path %v, got %v",
				testIPHealthcheckPath1,
				eip.Healthcheck.Path)
		}
		if eip.Healthcheck.Interval != testIPHealthcheckInterval1 {
			return fmt.Errorf("expected IP healthcheck interval %v, got %v",
				testIPHealthcheckInterval1,
				eip.Healthcheck.Interval)
		}
		if eip.Healthcheck.Timeout != testIPHealthcheckTimeout1 {
			return fmt.Errorf("expected IP healthcheck timeout %v, got %v",
				testIPHealthcheckTimeout1,
				eip.Healthcheck.Timeout)
		}
		if eip.Healthcheck.StrikesOk != testIPHealthcheckStrikesOk1 {
			return fmt.Errorf("expected IP healthcheck strikes-ok %v, got %v",
				testIPHealthcheckStrikesOk1,
				eip.Healthcheck.StrikesOk)
		}
		if eip.Healthcheck.StrikesFail != testIPHealthcheckStrikesFail1 {
			return fmt.Errorf("expected IP healthcheck strikes-fail %v, got %v",
				testIPHealthcheckStrikesFail1,
				eip.Healthcheck.StrikesFail)
		}

		return nil
	}
}

func testAccCheckIPAddressUpdate(eip *egoscale.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if eip.IPAddress == nil {
			return errors.New("IP address is nil")
		}

		if eip.Healthcheck == nil {
			return errors.New("IP healthcheck is nil")
		}
		if eip.Healthcheck.Mode != testIPHealthcheckMode2 {
			return fmt.Errorf("expected IP healthcheck mode %v, got %v",
				testIPHealthcheckMode2,
				eip.Healthcheck.Mode)
		}
		if eip.Healthcheck.Port != testIPHealthcheckPort2 {
			return fmt.Errorf("expected IP healthcheck port %v, got %v",
				testIPHealthcheckPort2,
				eip.Healthcheck.Port)
		}
		if eip.Healthcheck.Path != testIPHealthcheckPath2 {
			return fmt.Errorf("expected IP healthcheck path %v, got %v",
				testIPHealthcheckPath2,
				eip.Healthcheck.Path)
		}
		if eip.Healthcheck.Interval != testIPHealthcheckInterval2 {
			return fmt.Errorf("expected IP healthcheck interval %v, got %v",
				testIPHealthcheckInterval2,
				eip.Healthcheck.Interval)
		}
		if eip.Healthcheck.Timeout != testIPHealthcheckTimeout2 {
			return fmt.Errorf("expected IP healthcheck timeout %v, got %v",
				testIPHealthcheckTimeout2,
				eip.Healthcheck.Timeout)
		}
		if eip.Healthcheck.StrikesOk != testIPHealthcheckStrikesOk2 {
			return fmt.Errorf("expected IP healthcheck strikes-ok %v, got %v",
				testIPHealthcheckStrikesOk2,
				eip.Healthcheck.StrikesOk)
		}
		if eip.Healthcheck.StrikesFail != testIPHealthcheckStrikesFail2 {
			return fmt.Errorf("expected IP healthcheck strikes-fail %v, got %v",
				testIPHealthcheckStrikesFail2,
				eip.Healthcheck.StrikesFail)
		}

		return nil
	}
}

func testAccCheckIPAddressAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_ipaddress" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckIPAddressDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_ipaddress" {
			continue
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		key := &egoscale.IPAddress{
			ID:        id,
			IsElastic: true,
		}
		_, err = client.Get(key)
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
		return errors.New("IP address still exists")
	}
	return nil
}

var testAccIPAddressConfigCreate = fmt.Sprintf(`
resource "exoscale_ipaddress" "eip" {
  zone = %q
  description = "IPAdress 1"
  healthcheck_mode = "%s"
  healthcheck_port = %d
  healthcheck_path = "%s"
  healthcheck_interval = %d
  healthcheck_timeout = %d
  healthcheck_strikes_ok = %d
  healthcheck_strikes_fail = %d
  tags = {
    test = "acceptance"
  }
}
`,
	defaultExoscaleZone,
	testIPHealthcheckMode1,
	testIPHealthcheckPort1,
	testIPHealthcheckPath1,
	testIPHealthcheckInterval1,
	testIPHealthcheckTimeout1,
	testIPHealthcheckStrikesOk1,
	testIPHealthcheckStrikesFail1,
)

var testAccIPAddressConfigUpdate = fmt.Sprintf(`
resource "exoscale_ipaddress" "eip" {
  zone = %q
  description = "IPAdress 1 updated!"
  healthcheck_mode = "%s"
  healthcheck_port = %d
  healthcheck_path = "%s"
  healthcheck_interval = %d
  healthcheck_timeout = %d
  healthcheck_strikes_ok = %d
  healthcheck_strikes_fail = %d
}
`,
	defaultExoscaleZone,
	testIPHealthcheckMode2,
	testIPHealthcheckPort2,
	testIPHealthcheckPath2,
	testIPHealthcheckInterval2,
	testIPHealthcheckTimeout2,
	testIPHealthcheckStrikesOk2,
	testIPHealthcheckStrikesFail2,
)
