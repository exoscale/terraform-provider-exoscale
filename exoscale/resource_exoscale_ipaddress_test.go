package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceIPAddressZoneName                            = testZoneName
	testAccResourceIPAddressDescription                         = testDescription
	testAccResourceIPAddressHealthcheckMode                     = "http"
	testAccResourceIPAddressHealthcheckPort               int64 = 80
	testAccResourceIPAddressHealthcheckPortUpdated        int64 = 8000
	testAccResourceIPAddressHealthcheckPath                     = "/health"
	testAccResourceIPAddressHealthcheckPathUpdated              = "/healthz"
	testAccResourceIPAddressHealthcheckInterval           int64 = 10
	testAccResourceIPAddressHealthcheckIntervalUpdated    int64 = 5
	testAccResourceIPAddressHealthcheckTimeout            int64 = 5
	testAccResourceIPAddressHealthcheckTimeoutUpdated     int64 = 2
	testAccResourceIPAddressHealthcheckStrikesOk          int64 = 1
	testAccResourceIPAddressHealthcheckStrikesOkUpdated   int64 = 2
	testAccResourceIPAddressHealthcheckStrikesFail        int64 = 2
	testAccResourceIPAddressHealthcheckStrikesFailUpdated int64 = 3

	testAccIPAddressConfigCreate = fmt.Sprintf(`
resource "exoscale_ipaddress" "eip" {
  zone = "%s"
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
		testAccResourceIPAddressZoneName,
		testAccResourceIPAddressHealthcheckMode,
		testAccResourceIPAddressHealthcheckPort,
		testAccResourceIPAddressHealthcheckPath,
		testAccResourceIPAddressHealthcheckInterval,
		testAccResourceIPAddressHealthcheckTimeout,
		testAccResourceIPAddressHealthcheckStrikesOk,
		testAccResourceIPAddressHealthcheckStrikesFail,
	)

	testAccIPAddressConfigUpdate = fmt.Sprintf(`
resource "exoscale_ipaddress" "eip" {
  zone = "%s"
  description = "%s"
  healthcheck_mode = "%s"
  healthcheck_port = %d
  healthcheck_path = "%s"
  healthcheck_interval = %d
  healthcheck_timeout = %d
  healthcheck_strikes_ok = %d
  healthcheck_strikes_fail = %d
}
`,
		testAccResourceIPAddressZoneName,
		testAccResourceIPAddressDescription,
		testAccResourceIPAddressHealthcheckMode,
		testAccResourceIPAddressHealthcheckPortUpdated,
		testAccResourceIPAddressHealthcheckPathUpdated,
		testAccResourceIPAddressHealthcheckIntervalUpdated,
		testAccResourceIPAddressHealthcheckTimeoutUpdated,
		testAccResourceIPAddressHealthcheckStrikesOkUpdated,
		testAccResourceIPAddressHealthcheckStrikesFailUpdated,
	)
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
						"healthcheck_mode":         ValidateString(testAccResourceIPAddressHealthcheckMode),
						"healthcheck_port":         ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckPort)),
						"healthcheck_path":         ValidateString(testAccResourceIPAddressHealthcheckPath),
						"healthcheck_interval":     ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckInterval)),
						"healthcheck_timeout":      ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckTimeout)),
						"healthcheck_strikes_ok":   ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckStrikesOk)),
						"healthcheck_strikes_fail": ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckStrikesFail)),
					}),
				),
			},
			{
				Config: testAccIPAddressConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPAddressExists("exoscale_ipaddress.eip", eip),
					testAccCheckIPAddressUpdate(eip),
					testAccCheckIPAddressAttributes(testAttrs{
						"description":              ValidateString(testAccResourceIPAddressDescription),
						"healthcheck_mode":         ValidateString(testAccResourceIPAddressHealthcheckMode),
						"healthcheck_port":         ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckPortUpdated)),
						"healthcheck_path":         ValidateString(testAccResourceIPAddressHealthcheckPathUpdated),
						"healthcheck_interval":     ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckIntervalUpdated)),
						"healthcheck_timeout":      ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckTimeoutUpdated)),
						"healthcheck_strikes_ok":   ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckStrikesOkUpdated)),
						"healthcheck_strikes_fail": ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckStrikesFailUpdated)),
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
							"description":              ValidateString(testAccResourceIPAddressDescription),
							"healthcheck_mode":         ValidateString(testAccResourceIPAddressHealthcheckMode),
							"healthcheck_port":         ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckPortUpdated)),
							"healthcheck_path":         ValidateString(testAccResourceIPAddressHealthcheckPathUpdated),
							"healthcheck_interval":     ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckIntervalUpdated)),
							"healthcheck_timeout":      ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckTimeoutUpdated)),
							"healthcheck_strikes_ok":   ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckStrikesOkUpdated)),
							"healthcheck_strikes_fail": ValidateString(fmt.Sprint(testAccResourceIPAddressHealthcheckStrikesFailUpdated)),
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
		if eip.Healthcheck.Mode != testAccResourceIPAddressHealthcheckMode {
			return fmt.Errorf("expected IP healthcheck mode %v, got %v",
				testAccResourceIPAddressHealthcheckMode,
				eip.Healthcheck.Mode)
		}
		if eip.Healthcheck.Port != testAccResourceIPAddressHealthcheckPort {
			return fmt.Errorf("expected IP healthcheck port %v, got %v",
				testAccResourceIPAddressHealthcheckPort,
				eip.Healthcheck.Port)
		}
		if eip.Healthcheck.Path != testAccResourceIPAddressHealthcheckPath {
			return fmt.Errorf("expected IP healthcheck path %v, got %v",
				testAccResourceIPAddressHealthcheckPath,
				eip.Healthcheck.Path)
		}
		if eip.Healthcheck.Interval != testAccResourceIPAddressHealthcheckInterval {
			return fmt.Errorf("expected IP healthcheck interval %v, got %v",
				testAccResourceIPAddressHealthcheckInterval,
				eip.Healthcheck.Interval)
		}
		if eip.Healthcheck.Timeout != testAccResourceIPAddressHealthcheckTimeout {
			return fmt.Errorf("expected IP healthcheck timeout %v, got %v",
				testAccResourceIPAddressHealthcheckTimeout,
				eip.Healthcheck.Timeout)
		}
		if eip.Healthcheck.StrikesOk != testAccResourceIPAddressHealthcheckStrikesOk {
			return fmt.Errorf("expected IP healthcheck strikes-ok %v, got %v",
				testAccResourceIPAddressHealthcheckStrikesOk,
				eip.Healthcheck.StrikesOk)
		}
		if eip.Healthcheck.StrikesFail != testAccResourceIPAddressHealthcheckStrikesFail {
			return fmt.Errorf("expected IP healthcheck strikes-fail %v, got %v",
				testAccResourceIPAddressHealthcheckStrikesFail,
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
		if eip.Healthcheck.Mode != testAccResourceIPAddressHealthcheckMode {
			return fmt.Errorf("expected IP healthcheck mode %v, got %v",
				testAccResourceIPAddressHealthcheckMode,
				eip.Healthcheck.Mode)
		}
		if eip.Healthcheck.Port != testAccResourceIPAddressHealthcheckPortUpdated {
			return fmt.Errorf("expected IP healthcheck port %v, got %v",
				testAccResourceIPAddressHealthcheckPortUpdated,
				eip.Healthcheck.Port)
		}
		if eip.Healthcheck.Path != testAccResourceIPAddressHealthcheckPathUpdated {
			return fmt.Errorf("expected IP healthcheck path %v, got %v",
				testAccResourceIPAddressHealthcheckPathUpdated,
				eip.Healthcheck.Path)
		}
		if eip.Healthcheck.Interval != testAccResourceIPAddressHealthcheckIntervalUpdated {
			return fmt.Errorf("expected IP healthcheck interval %v, got %v",
				testAccResourceIPAddressHealthcheckIntervalUpdated,
				eip.Healthcheck.Interval)
		}
		if eip.Healthcheck.Timeout != testAccResourceIPAddressHealthcheckTimeoutUpdated {
			return fmt.Errorf("expected IP healthcheck timeout %v, got %v",
				testAccResourceIPAddressHealthcheckTimeoutUpdated,
				eip.Healthcheck.Timeout)
		}
		if eip.Healthcheck.StrikesOk != testAccResourceIPAddressHealthcheckStrikesOkUpdated {
			return fmt.Errorf("expected IP healthcheck strikes-ok %v, got %v",
				testAccResourceIPAddressHealthcheckStrikesOkUpdated,
				eip.Healthcheck.StrikesOk)
		}
		if eip.Healthcheck.StrikesFail != testAccResourceIPAddressHealthcheckStrikesFailUpdated {
			return fmt.Errorf("expected IP healthcheck strikes-fail %v, got %v",
				testAccResourceIPAddressHealthcheckStrikesFailUpdated,
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
