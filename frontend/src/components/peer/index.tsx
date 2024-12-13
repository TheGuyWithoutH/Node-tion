"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { z } from "zod"

import {AddPeer as Add_Peer, GetAddress} from "../../../wailsjs/go/impl/node"

import { toast } from "@/hooks/use-toast"
import { Button } from "@/components/ui/button"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

import { Input } from "@/components/ui/input"
import { Toaster } from "../ui/toaster"
import { useEffect, useState } from "react"

const FormSchema = z.object({
  ipAddress: z.string().regex(/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/, {
    message: "Please enter a valid IP address.",
  }),
})

export default function AddPeer() {
  const form = useForm<z.infer<typeof FormSchema>>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      ipAddress: "",
    },
  })

  const [address, setAddress] = useState<string>("")

  useEffect(() => {
    GetAddress().then((addr) => {
      console.log(addr)
      setAddress(addr)
    }).catch((err) => {
      console.log(err)
    })
  }, [])

  function onSubmit(data: z.infer<typeof FormSchema>) {
    toast({
      title: "Added Peer",
      description: "You can now start editting the document together in the editor",
    })
    const ip = [data.ipAddress]
    Add_Peer(ip)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="w-2/3 space-y-6">
        <FormField
          control={form.control}
          name="ipAddress"
          render={({ field }) => (
            <FormItem>
              <FormLabel>IP Address</FormLabel>
              <FormControl>
                <Input placeholder="192.168.0.1" {...field} />
              </FormControl>
              <FormDescription>
                Enter the IP address of the peer you want to add. You are {address}.
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit">Add Peer</Button>
        <Toaster />
      </form>
    </Form>
  )
}
