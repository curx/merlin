package main

import (
	"errors"
	"regexp"

	"strconv"

	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/sky-uk/merlin/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var serviceCmd = &cobra.Command{
	Use:   "service [add|edit]",
	Short: "Add or edit a virtual service",
}

var hostPortRegex = regexp.MustCompile(`^([^:]+):(\d+)$`)

func validIDProtocolHostPort(_ *cobra.Command, args []string) error {
	if len(args) != 3 {
		return errors.New("requires three arguments")
	}
	b := []byte(args[2])
	if !hostPortRegex.Match(b) {
		return errors.New("must be ip:port")
	}
	return nil
}

var addServiceCmd = &cobra.Command{
	Use:   "add [id] [protocol] [host:port]",
	Short: "Add a virtual service",
	Args:  validIDProtocolHostPort,
	RunE:  addService,
}

var editServiceCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit a virtual service",
	Args:  cobra.ExactArgs(1),
	RunE:  editService,
}

var deleteServiceCmd = &cobra.Command{
	Use:   "del [id]",
	Short: "Delete a virtual service",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteService,
}

var (
	scheduler  string
	schedFlags []string
)

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(addServiceCmd)
	serviceCmd.AddCommand(editServiceCmd)
	serviceCmd.AddCommand(deleteServiceCmd)

	for _, f := range []*pflag.FlagSet{addServiceCmd.Flags(), editServiceCmd.Flags()} {
		f.StringVarP(&scheduler, "scheduler", "s", "", "scheduler for new connections")
		f.StringSliceVarP(&schedFlags, "sched-flags", "b", nil, "scheduler flags")
	}
}

func serviceFromFlags(id string) *types.VirtualService {
	return &types.VirtualService{
		Id: id,
		Config: &types.VirtualService_Config{
			Scheduler: scheduler,
			Flags:     schedFlags,
		},
	}
}

func addService(_ *cobra.Command, args []string) error {
	return client(func(c types.MerlinClient) error {
		svc := serviceFromFlags(args[0])

		proto, ok := types.Protocol_value[strings.ToUpper(args[1])]
		if !ok {
			return errors.New("unrecognized protocol")
		}
		matches := hostPortRegex.FindSubmatch([]byte(args[2]))
		host := string(matches[1])
		port, err := strconv.ParseUint(string(matches[2]), 10, 16)
		if err != nil {
			return fmt.Errorf("unable to parse port: %v", err)
		}

		svc.Key = &types.VirtualService_Key{
			Protocol: types.Protocol(proto),
			Ip:       host,
			Port:     uint32(port),
		}

		ctx, cancel := clientContext()
		defer cancel()
		_, err = c.CreateService(ctx, svc)
		return err
	})
}

func editService(_ *cobra.Command, args []string) error {
	return client(func(c types.MerlinClient) error {
		svc := serviceFromFlags(args[0])
		ctx, cancel := clientContext()
		defer cancel()
		_, err := c.UpdateService(ctx, svc)
		return err
	})
}

func deleteService(_ *cobra.Command, args []string) error {
	return client(func(c types.MerlinClient) error {
		id := wrappers.StringValue{Value: args[0]}
		ctx, cancel := clientContext()
		defer cancel()
		_, err := c.DeleteService(ctx, &id)
		return err
	})
}